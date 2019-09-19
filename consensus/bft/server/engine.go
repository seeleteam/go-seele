package server

import (
	"bytes"
	"errors"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
	lru "github.com/hashicorp/golang-lru"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/bft"
	bftCore "github.com/seeleteam/go-seele/consensus/bft/core"
	"github.com/seeleteam/go-seele/consensus/bft/verifier"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
)

const (
	checkpointInterval = 1024 // Height of blocks after which to save the vote snapshot to the database
	inmemorySnapshots  = 128  // Height of recent vote snapshots to keep in memory
	inmemoryPeers      = 40
	inmemoryMessages   = 1024
)

var (
	// errInvalidProposal is returned when a prposal is malformed.
	errInvalidProposal = errors.New("invalid proposal")
	// errInvalidSignature is returned when given signature is not signed by given
	// address.
	errInvalidSignature = errors.New("invalid signature")
	// errUnknownBlock is returned when the list of verifiers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")
	// errUnauthorized is returned if a header is signed by a non authorized entity.
	errUnauthorized = errors.New("unauthorized")
	// errInvalidDifficulty is returned if the difficulty of a block is not 1
	errInvalidDifficulty = errors.New("invalid difficulty")
	// errInvalidExtraDataFormat is returned when the extra data format is incorrect
	errInvalidExtraDataFormat = errors.New("invalid extra data format")
	// errBFTConsensus is returned if a block's consensus mismatch BFT
	errBFTConsensus = errors.New("mismatch BFT Consensus")
	// errInvalidNonce is returned if a block's nonce is invalid
	errInvalidNonce = errors.New("invalid nonce")
	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")
	// errInconsistentValidatorSet is returned if the verifier set is inconsistent
	errInconsistentValidatorSet = errors.New("non empty uncle hash")
	// errInvalidTimestamp is returned if the timestamp of a block is lower than the previous block's timestamp + the minimum block period.
	errInvalidTimestamp = errors.New("invalid timestamp")
	// errInvalidVotingChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errInvalidVotingChain = errors.New("invalid voting chain")
	// errInvalidVote is returned if a nonce value is something else that the two
	// allowed constants of 0x00..0 or 0xff..f.
	errInvalidVote = errors.New("vote nonce not 0x00..0 or 0xff..f")
	// errInvalidCommittedSeals is returned if the committed seal is not signed by any of parent verifiers.
	errInvalidCommittedSeals = errors.New("invalid committed seals")
	// errEmptyCommittedSeals is returned if the field of committed seals is zero.
	errEmptyCommittedSeals = errors.New("zero committed seals")
	// errMismatchTxhashes is returned if the TxHash in header is mismatch.
	errMismatchTxhashes = errors.New("mismatch transcations hashes")
)

var (
	defaultDifficulty = big.NewInt(1)
	now               = time.Now

	nonceAuthVote = hexutil.MustHexToBytes("0xffffffffffffffff") // Magic nonce number to vote on adding a new verifier
	nonceDropVote = hexutil.MustHexToBytes("0x0000000000000000") // Magic nonce number to vote on removing a verifier.

	inmemoryAddresses  = 20 // Height of recent addresses from extractAccount
	recentAddresses, _ = lru.NewARC(inmemoryAddresses)
)

func (s *server) Prepare(chain consensus.ChainReader, header *types.BlockHeader) error {

	//1. setup some unused field
	header.Creator = common.Address{}
	header.Witness = make([]byte, bft.WitnessSize)
	header.Consensus = types.BftConsensus
	header.Difficulty = defaultDifficulty

	// 2. copy parent extra data as the header extra data
	height := header.Height
	parent := chain.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return consensus.ErrBlockInvalidParentHash
	}
	// voting snapshot
	snap, err := s.snapshot(chain, height-1, header.PreviousBlockHash, nil)
	if err != nil {
		return err
	}

	//get valid candidate list
	s.candidatesLock.RLock()
	var addrs []common.Address
	var auths []bool
	for addr, auth := range s.candidates {
		if snap.checkVote(addr, auth) {
			addrs = append(addrs, addr)
			auths = append(auths, auth)
		}
	}
	s.candidatesLock.RUnlock()

	// pick one candidate randomly
	if len(addrs) > 0 {
		index := rand.Intn(len(addrs))
		header.Creator = addrs[index]
		if auths[index] {
			copy(header.Witness[:], nonceAuthVote) //
		} else {
			copy(header.Witness[:], nonceDropVote) //
		}
	}

	// add verifiers in snapshot to extraData's verifiers section
	extra, err := prepareExtra(header, snap.verifiers()) // TODO implement prepareExtra
	if err != nil {
		return err
	}
	header.ExtraData = extra

	// set timeStamp at header
	header.CreateTimestamp = new(big.Int).Add(parent.CreateTimestamp, new(big.Int).SetUint64(s.config.BlockPeriod))
	// but if creatTimestamp is smaller than current. set to current!
	if header.CreateTimestamp.Int64() < time.Now().Unix() {
		header.CreateTimestamp = big.NewInt(time.Now().Unix())
	}

	// finish all process
	return nil
}

func (s *server) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}, results chan<- *types.Block) error {
	block, err := s.SealResult(chain, block, stop)
	results <- block
	return err
}

// Seal generates a new block for the given input block with the local miner's
// seal place on top.
func (s *server) SealResult(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	// update the block header timestamp and signature and propose the block to core engine
	header := block.Header
	number := header.Height

	// Bail out if we're unauthorized to sign a block
	snap, err := s.snapshot(chain, number-1, header.PreviousBlockHash, nil)
	if err != nil {
		return nil, err
	}
	if _, v := snap.VerSet.GetByAddress(s.address); v == nil {
		return nil, errUnauthorized
	}

	parent := chain.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return nil, consensus.ErrBlockInvalidParentHash
	}
	block, err = s.updateBlock(parent, block) // TODO implement
	if err != nil {
		return nil, err
	}

	// wait for the timestamp of header, use this to adjust the block period
	delay := time.Unix(block.Header.CreateTimestamp.Int64(), 0).Sub(now())
	select {
	case <-time.After(delay):
	case <-stop:
		return nil, nil
	}

	// get the proposed block hash and clear it if the seal() is completed.
	s.sealMu.Lock()
	s.proposedBlockHash = block.Hash()
	clear := func() {
		s.proposedBlockHash = common.Hash{}
		s.sealMu.Unlock()
	}
	defer clear()

	// post block into Istanbul engine
	go s.EventMux().Post(bft.RequestEvent{
		Proposal: block,
	})

	for {
		select {
		case result := <-s.commitCh:
			// if the block hash and the hash from channel are the same,
			// return the result. Otherwise, keep waiting the next hash.
			if block.Hash() == result.Hash() {
				return result, nil
			}
		case <-stop:
			return nil, nil
		}
	}
}

func (s *server) VerifyHeader(chain consensus.ChainReader, header *types.BlockHeader) error {
	return s.verifyHeader(chain, header, nil)
}

// verifyHeader
// consensus-createTime- extraData-the block is not voting on add or remove one verifier-difficulty
func (s *server) verifyHeader(chain consensus.ChainReader, header *types.BlockHeader, parents []*types.BlockHeader) error {
	if header.Consensus != types.BftConsensus {
		return errBFTConsensus
	}
	if header.CreateTimestamp.Cmp(big.NewInt(now().Unix())) > 0 {
		return consensus.ErrBlockCreateTimeOld
	}
	if _, err := types.ExtractBftExtra(header); err != nil {
		return errInvalidExtraDataFormat
	}
	if header.Height != 0 && !bytes.Equal(header.Witness[:], nonceAuthVote) && !bytes.Equal(header.Witness[:], nonceDropVote) {
		return errInvalidNonce
	}
	if header.Difficulty == nil || header.Difficulty.Cmp(defaultDifficulty) != 0 {
		return errInvalidDifficulty
	}
	return s.verifyCascadingFields(chain, header, parents)
}

func (s *server) verifyCascadingFields(chain consensus.ChainReader, header *types.BlockHeader, parents []*types.BlockHeader) error {
	number := header.Height
	if number == 0 {
		return nil
	}

	// Ensure that the block's timestamp isn't too close to it's parent
	var parent *types.BlockHeader
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeaderByHash(header.PreviousBlockHash)
	}
	if parent.CreateTimestamp.Uint64()+s.config.BlockPeriod > header.CreateTimestamp.Uint64() {
		return errInvalidTimestamp
	}
	// verify verify extraData. Verifiers in snapshot and extraData should be the same
	snap, err := s.snapshot(chain, number-1, header.PreviousBlockHash, parents) //TODO implement snapshot() in snapshot.go
	if err != nil {
		return err
	}
	verifiers := make([]byte, len(snap.verifiers())*common.AddressLen) //TODO implement verifiers() in snapshot.go
	for i, verifier := range snap.verifiers() {
		copy(verifiers[i*common.AddressLen:], verifier[:])
	}
	if err := s.verifySigner(chain, header, parents); err != nil {
		return err
	}
	// verify committed seals
	return s.verifyCommittedSeals(chain, header, parents)
}

// verifyCommittedSeals checks whether every committed seal is signed by one of the parent's validators
func (s *server) verifyCommittedSeals(chain consensus.ChainReader, header *types.BlockHeader, parents []*types.BlockHeader) error {
	number := header.Height
	if number == 0 {
		return nil
	}
	snap, err := s.snapshot(chain, number-1, header.PreviousBlockHash, parents)
	if err != nil {
		return err
	}
	extra, err := types.ExtractBftExtra(header)
	if err != nil {
		return err
	}
	if len(extra.CommittedSeal) == 0 {
		return errEmptyCommittedSeals
	}
	verifiers := snap.VerSet.Copy() //TODO
	validSealCount := 0
	proposalSeal := bftCore.PrepareCommitedSeal(header.Hash())
	// 1. get committed seals from current header
	for _, seal := range extra.CommittedSeal {
		addr, err := bft.GetSignatureAddress(proposalSeal, seal)
		if err != nil {
			s.log.Error("not a valid address, err", err)
			return errInvalidSignature
		}
		if verifiers.RemoveVerifier(addr) { //TODO
			validSealCount += 1
		} else {
			return errInvalidCommittedSeals
		}
	}
	// 2. The length of validSeal should be larger than number of faulty node + 1
	if validSealCount <= 2*snap.VerSet.F() {
		return errInvalidCommittedSeals
	}
	return nil
}

func (s *server) verifySigner(chain consensus.ChainReader, header *types.BlockHeader, parents []*types.BlockHeader) error {
	number := header.Height
	if number == 0 {
		return errUnknownBlock
	}
	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := s.snapshot(chain, number-1, header.PreviousBlockHash, parents)
	if err != nil {
		return err
	}

	// resolve the authorization key and check against signers
	signer, err := extractAccount(header)
	if err != nil {
		return err
	}

	// Signer should be in the validator set of previous block's extraData.
	if _, v := snap.VerSet.GetByAddress(signer); v == nil {
		return errUnauthorized
	}
	return nil
}

// Author retrieves the Seele address of the account that minted the given
// block, which may be different from the header's coinbase if a consensus
// engine is based on signatures.
func (s *server) Creator(header *types.BlockHeader) (common.Address, error) {
	return extractAccount(header)
}
func extractAccount(header *types.BlockHeader) (common.Address, error) {
	hash := header.Hash()
	if addr, ok := recentAddresses.Get(hash); ok {
		return addr.(common.Address), nil
	}
	bftExtra, err := types.ExtractBftExtra(header) // TODO!!!! redifine ExtractBftExtra
	if err != nil {
		return common.Address{}, err
	}
	addr, err := bft.GetSignatureAddress(sigHash(header).Bytes(), bftExtra.Seal)
	if err != nil {
		return addr, err
	}
	recentAddresses.Add(hash, addr)
	return addr, nil
}

// FIXME: Need to update this for Istanbul
// sigHash returns the hash which is used as input for the Istanbul
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func sigHash(header *types.BlockHeader) (hash common.Hash) {
	h := types.IstanbulFilteredHeader(header, false) //TODO
	return crypto.MustHash(h)
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (ser *server) snapshot(chain consensus.ChainReader, height uint64, hash common.Hash, parents []*types.BlockHeader) (*Snapshot, error) {
	// Search for a snapshot in memory or on disk for checkpoints
	var (
		headers []*types.BlockHeader
		snap    *Snapshot
	)
	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := ser.recents.Get(hash); ok {
			snap = s.(*Snapshot)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if height%checkpointInterval == 0 {
			if s, err := loadSnapshot(ser.config.Epoch, ser.db, hash); err == nil {
				ser.log.Debug("Loaded voting snapshot form disk. height: %d. hash %s", height, hash)
				snap = s
				break
			}
		}
		// If we're at block zero, make a snapshot
		if height == 0 {
			genesis := chain.GetHeaderByHeight(0)
			if err := ser.VerifyHeader(chain, genesis); err != nil {
				return nil, err
			}
			bftExtra, err := types.ExtractBftExtra(genesis)
			if err != nil {
				return nil, err
			}
			snap = newSnapshot(ser.config.Epoch, 0, genesis.Hash(), verifier.NewVerifierSet(bftExtra.Verifiers, ser.config.ProposerPolicy))
			if err := snap.store(ser.db); err != nil {
				return nil, err
			}
			ser.log.Debug("Stored genesis voting snapshot to disk")
			break
		}
		// No snapshot for this header, gather the header and move backward
		var header *types.BlockHeader
		if len(parents) > 0 {
			// If we have explicit parents, pick from there (enforced)
			header = parents[len(parents)-1]
			if header.Hash() != hash || header.Height != height {
				return nil, consensus.ErrBlockInvalidParentHash
			}
			parents = parents[:len(parents)-1]
		} else {
			// No explicit parents (or no more left), reach out to the database
			header = chain.GetHeaderByHash(hash)
			if header == nil {
				return nil, consensus.ErrBlockInvalidParentHash
			}
		}
		headers = append(headers, header)
		height, hash = height-1, header.PreviousBlockHash
	}
	// Previous snapshot found, apply any pending headers on top of it
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}
	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}
	ser.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Height%checkpointInterval == 0 && len(headers) > 0 {
		if err = snap.store(ser.db); err != nil {
			return nil, err
		}
		ser.log.Debug("Stored voting snapshot to disk. height %d. hash %s", snap.Height, snap.Hash)
	}
	return snap, err
}

// prepareExtra returns a extra-data of the given header and validators
func prepareExtra(header *types.BlockHeader, vers []common.Address) ([]byte, error) {
	var buf bytes.Buffer

	// compensate the lack bytes if header.Extra is not enough BftExtraVanity bytes.
	if len(header.ExtraData) < types.BftExtraVanity { //here we use BftExtraVanity (32-bit fixed length)
		header.ExtraData = append(header.ExtraData, bytes.Repeat([]byte{0x00}, types.BftExtraVanity-len(header.ExtraData))...)
	}
	buf.Write(header.ExtraData[:types.BftExtraVanity])

	bfte := &types.BftExtra{ // we share the BftExtra struct
		Verifiers:     vers,
		Seal:          []byte{},
		CommittedSeal: [][]byte{},
	}

	payload, err := rlp.EncodeToBytes(&bfte)
	if err != nil {
		return nil, err
	}

	return append(buf.Bytes(), payload...), nil
}

// update timestamp and signature of the block based on its number of transactions
func (s *server) updateBlock(parent *types.BlockHeader, block *types.Block) (*types.Block, error) {
	header := block.Header
	// sign the hash
	seal, err := s.Sign(sigHash(header).Bytes())
	if err != nil {
		return nil, err
	}

	err = writeSeal(header, seal)
	if err != nil {
		return nil, err
	}

	return block.WithSeal(header), nil
}

// writeSeal writes the extra-data field of the given header with the given seals.
// suggest to rename to writeSeal.
func writeSeal(h *types.BlockHeader, seal []byte) error {
	if len(seal)%types.BftExtraSeal != 0 {
		return errInvalidSignature
	}

	bftExtra, err := types.ExtractBftExtra(h)
	if err != nil {
		return err
	}

	bftExtra.Seal = seal
	payload, err := rlp.EncodeToBytes(&bftExtra)
	if err != nil {
		return err
	}

	h.ExtraData = append(h.ExtraData[:types.BftExtraVanity], payload...)
	return nil
}

func writeCommittedSeals(h *types.BlockHeader, committedSeals [][]byte) error {
	if len(committedSeals) == 0 {
		return errInvalidCommittedSeals
	}

	for _, seal := range committedSeals {
		if len(seal) != types.BftExtraSeal { // TODO change types
			return errInvalidCommittedSeals
		}
	}

	bftExtra, err := types.ExtractBftExtra(h) // TODO change types
	if err != nil {
		return err
	}

	bftExtra.CommittedSeal = make([][]byte, len(committedSeals))
	copy(bftExtra.CommittedSeal, committedSeals) // TODO change types

	payload, err := rlp.EncodeToBytes(&bftExtra)
	if err != nil {
		return err
	}

	h.ExtraData = append(h.ExtraData[:types.BftExtraVanity], payload...) // TODO change types
	return nil
}
