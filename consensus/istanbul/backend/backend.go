/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package backend

import (
	"crypto/ecdsa"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/hashicorp/golang-lru"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/istanbul"
	istanbulCore "github.com/seeleteam/go-seele/consensus/istanbul/core"
	"github.com/seeleteam/go-seele/consensus/istanbul/validator"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/log"
)

const (
	// fetcherID is the ID indicates the block is from Istanbul engine
	fetcherID = "istanbul"
)

// New creates an Ethereum backend for Istanbul core engine.
func New(config *istanbul.Config, privateKey *ecdsa.PrivateKey, db database.Database) consensus.Istanbul {
	// Allocate the snapshot caches and create the engine
	recents, _ := lru.NewARC(inmemorySnapshots)
	recentMessages, _ := lru.NewARC(inmemoryPeers)
	knownMessages, _ := lru.NewARC(inmemoryMessages)
	backend := &backend{
		config:           config,
		istanbulEventMux: new(event.TypeMux),
		privateKey:       privateKey,
		address:          crypto.PubkeyToAddress(privateKey.PublicKey),
		logger:           log.GetLogger("ibft"),
		db:               db,
		commitCh:         make(chan *types.Block, 1),
		recents:          recents,
		candidates:       make(map[common.Address]bool),
		coreStarted:      false,
		recentMessages:   recentMessages,
		knownMessages:    knownMessages,
	}
	backend.core = istanbulCore.New(backend, backend.config)
	return backend
}

// ----------------------------------------------------------------------------

type backend struct {
	config           *istanbul.Config
	istanbulEventMux *event.TypeMux
	privateKey       *ecdsa.PrivateKey
	address          common.Address
	core             istanbulCore.Engine
	logger           *log.SeeleLog
	db               database.Database
	chain            consensus.ChainReader
	currentBlock     func() *types.Block
	hasBadBlock      func(hash common.Hash) bool

	// the channels for istanbul engine notifications
	commitCh          chan *types.Block
	proposedBlockHash common.Hash
	sealMu            sync.Mutex
	coreStarted       bool
	coreMu            sync.RWMutex

	// Current list of candidates we are pushing
	candidates map[common.Address]bool
	// Protects the signer fields
	candidatesLock sync.RWMutex
	// Snapshots for recent block to speed up reorgs
	recents *lru.ARCCache

	// event subscription for ChainHeadEvent event
	broadcaster consensus.Broadcaster

	recentMessages *lru.ARCCache // the cache of peer's messages
	knownMessages  *lru.ARCCache // the cache of self messages
}

// Address implements istanbul.Backend.Address
func (sb *backend) Address() common.Address {
	return sb.address
}

// Validators implements istanbul.Backend.Validators
func (sb *backend) Validators(proposal istanbul.Proposal) istanbul.ValidatorSet {
	return sb.getValidators(proposal.Height(), proposal.Hash())
}

// Broadcast implements istanbul.Backend.Broadcast
func (sb *backend) Broadcast(valSet istanbul.ValidatorSet, payload []byte) error {
	// send to others
	sb.Gossip(valSet, payload)
	// send to self
	msg := istanbul.MessageEvent{
		Payload: payload,
	}
	go sb.istanbulEventMux.Post(msg)
	return nil
}

// Broadcast implements istanbul.Backend.Gossip
func (sb *backend) Gossip(valSet istanbul.ValidatorSet, payload []byte) error {
	hash := istanbul.RLPHash(payload)
	sb.knownMessages.Add(hash, true)

	targets := make(map[common.Address]bool)
	for _, val := range valSet.List() {
		if val.Address() != sb.Address() {
			targets[val.Address()] = true
		}
	}

	if sb.broadcaster != nil && len(targets) > 0 {
		ps := sb.broadcaster.FindPeers(targets)
		for addr, p := range ps {
			ms, ok := sb.recentMessages.Get(addr)
			var m *lru.ARCCache
			if ok {
				m, _ = ms.(*lru.ARCCache)
				if _, k := m.Get(hash); k {
					// This peer had this event, skip it
					continue
				}
			} else {
				m, _ = lru.NewARC(inmemoryMessages)
			}

			m.Add(hash, true)
			sb.recentMessages.Add(addr, m)

			go p.Send(istanbulMsg, payload)
		}
	}
	return nil
}

// Commit implements istanbul.Backend.Commit
func (sb *backend) Commit(proposal istanbul.Proposal, seals [][]byte) error {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		sb.logger.Error("Invalid proposal, %v", proposal)
		return errInvalidProposal
	}

	h := block.Header
	// Append seals into extra-data
	err := writeCommittedSeals(h, seals)
	if err != nil {
		return err
	}
	// update block's header
	block = block.WithSeal(h)

	sb.logger.Info("Committed", "address", sb.Address(), "hash", proposal.Hash(), "number", proposal.Height())
	// - if the proposed and committed blocks are the same, send the proposed hash
	//   to commit channel, which is being watched inside the engine.Seal() function.
	// - otherwise, we try to insert the block.
	// -- if success, the ChainHeadEvent event will be broadcasted, try to build
	//    the next block and the previous Seal() will be stopped.
	// -- otherwise, a error will be returned and a round change event will be fired.
	if sb.proposedBlockHash == block.Hash() {
		// feed block hash to Seal() and wait the Seal() result
		sb.commitCh <- block
		return nil
	}

	if sb.broadcaster != nil {
		sb.broadcaster.Enqueue(fetcherID, block)
	}
	return nil
}

// EventMux implements istanbul.Backend.EventMux
func (sb *backend) EventMux() *event.TypeMux {
	return sb.istanbulEventMux
}

// Verify implements istanbul.Backend.Verify
func (sb *backend) Verify(proposal istanbul.Proposal) (time.Duration, error) {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		sb.logger.Error("Invalid proposal, %v", proposal)
		return 0, errInvalidProposal
	}

	//// check bad block
	//if sb.HasBadProposal(block.Hash()) {
	//	return 0, core.ErrBlacklistedHash
	//}

	// check block body
	txnHash := types.MerkleRootHash(block.Transactions)
	if txnHash != block.Header.TxHash {
		return 0, errMismatchTxhashes
	}

	// verify the header of proposed block
	err := sb.VerifyHeader(sb.chain, block.Header)
	// ignore errEmptyCommittedSeals error because we don't have the committed seals yet
	if err == nil || err == errEmptyCommittedSeals {
		return 0, nil
	} else if err == consensus.ErrBlockCreateTimeOld {
		return time.Unix(block.Header.CreateTimestamp.Int64(), 0).Sub(now()), consensus.ErrBlockCreateTimeOld
	}
	return 0, err
}

// Sign implements istanbul.Backend.Sign
func (sb *backend) Sign(data []byte) ([]byte, error) {
	hashData := crypto.Keccak256([]byte(data))
	sign, err := crypto.Sign(sb.privateKey, hashData)
	return sign.Sig, err
}

// CheckSignature implements istanbul.Backend.CheckSignature
func (sb *backend) CheckSignature(data []byte, address common.Address, sig []byte) error {
	signer, err := istanbul.GetSignatureAddress(data, sig)
	if err != nil {
		sb.logger.Error("Failed to get signer address err %s", err)
		return err
	}
	// Compare derived addresses
	if signer != address {
		return errInvalidSignature
	}
	return nil
}

// HasPropsal implements istanbul.Backend.HashBlock
func (sb *backend) HasPropsal(hash common.Hash) bool {
	return sb.chain.GetHeaderByHash(hash) != nil
}

// GetProposer implements istanbul.Backend.GetProposer
func (sb *backend) GetProposer(height uint64) common.Address {
	if h := sb.chain.GetHeaderByHeight(height); h != nil {
		a, _ := sb.Author(h)
		return a
	}
	return common.Address{}
}

// ParentValidators implements istanbul.Backend.GetParentValidators
func (sb *backend) ParentValidators(proposal istanbul.Proposal) istanbul.ValidatorSet {
	if block, ok := proposal.(*types.Block); ok {
		return sb.getValidators(block.Height()-1, block.ParentHash())
	}
	return validator.NewSet(nil, sb.config.ProposerPolicy)
}

func (sb *backend) getValidators(number uint64, hash common.Hash) istanbul.ValidatorSet {
	snap, err := sb.snapshot(sb.chain, number, hash, nil)
	if err != nil {
		return validator.NewSet(nil, sb.config.ProposerPolicy)
	}
	return snap.ValSet
}

func (sb *backend) LastProposal() (istanbul.Proposal, common.Address) {
	block := sb.currentBlock()

	var proposer common.Address
	if block.Height() > 0 {
		var err error
		proposer, err = sb.Author(block.Header)
		if err != nil {
			sb.logger.Error("Failed to get block proposer", "err", err)
			return nil, common.Address{}
		}
	}

	// Return header only block here since we don't need block body
	return block, proposer
}

func (sb *backend) HasBadProposal(hash common.Hash) bool {
	if sb.hasBadBlock == nil {
		return false
	}
	return sb.hasBadBlock(hash)
}
