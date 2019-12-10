package server

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
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
	checkInterval     = 1024 // Height of blocks after which to save the vote snapshot to the database
	inmemorySnapshots = 128  // Height of recent vote snapshots to keep in memory
	inmemoryPeers     = 40   // peers of recent kept in memory
	inmemoryMessages  = 1024 // messages of recent kept in memory
)

var (
	// errInconsistentValidatorSet is returned if the verifier set is inconsistent
	errProposalInvalididatorSet = errors.New("non empty uncle hash")
	// errTimestampInvalid is returned if the timestamp of a block is lower than the previous block's timestamp + the minimum block period.
	errTimestampInvalid = errors.New("invalid timestamp")
	// errVotingChainInvalid is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errVotingChainInvalid = errors.New("invalid voting chain")
	// errUnauthorized is returned if a header is signed by a non authorized entity.
	errUnauthorized = errors.New("unauthorized")
	// errDifficultyInvalid is returned if the difficulty of a block is not 1
	errDifficultyInvalid = errors.New("invalid difficulty")
	// errExtraDataFormatInvalid is returned when the extra data format is incorrect
	errExtraDataFormatInvalid = errors.New("format of extra data is invalid")
	// errBFTConsensus is returned if a block's consensus mismatch BFT
	errBFTConsensus = errors.New("mismatch BFT Consensus")
	// errVoteInvalid is returned if a nonce value is something else that the two
	// allowed constants of 0x00..0 or 0xff..f.
	errVoteInvalid = errors.New("vote nonce not 0x00..0 or 0xff..f")
	// errCommittedSealsInvalid is returned if the committed seal is not signed by any of parent verifiers.
	errCommittedSealsInvalid = errors.New("committed seals are invalid")
	// errEmptyCommittedSeals is returned if the field of committed seals is zero.
	errEmptyCommittedSeals = errors.New("zero committed seals")
	// errMismatchTxhashes is returned if the TxHash in header is mismatch.
	errMismatchTxhashes = errors.New("mismatch transcations hashes")
	// errProposalInvalid is returned when a prposal is malformed.
	errProposalInvalid = errors.New("invalid proposal")
	// errInvalidSignature is returned when given signature is not signed by given
	// address.
	errInvalidSignature = errors.New("invalid signature")
	// errBlockUnknown is returned when the list of verifiers is requested for a block
	// that is not part of the local blockchain.
	errBlockUnknown = errors.New("unknown block")
	// errNonceInvalid is returned if a block's nonce is invalid
	errNonceInvalid = errors.New("invalid nonce")
	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")
)

var (
	defaultDifficulty = big.NewInt(1)
	now               = time.Now

	nonceAuthVote = hexutil.MustHexToBytes("0xffffffffffffffff") // Magic nonce number to vote on adding a new verifier
	nonceDropVote = hexutil.MustHexToBytes("0x0000000000000000") // Magic nonce number to vote on removing a verifier.

	inmemoryAddresses = 20 // Height of recent addresses from extractAccount
	cachedAddrs, _    = lru.NewARC(inmemoryAddresses)
)

// SealResult generates a new block for the given input block with the local miner's Seal.
func (s *server) SealResult(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	// update the block header timestamp and signature and propose the block to core engine
	header := block.Header
	number := header.Height

	// Bail out if we're unauthorized to sign a block
	s.log.Info("SealResult take a snapshot")
	snap, err := s.snapshot(chain, number-1, header.PreviousBlockHash, nil)
	if err != nil {
		return nil, err
	}
	// check whether self is authoried or not
	// Test Result return with VerSet:0xc000356640
	// s.log.Info("check s.address %+v in verset or not?", s.address)
	// s.log.Info("snap.VerSet %d verifiers, with snap.Verset %+v", snap.VerSet.Size(), snap.VerSet)

	// after mining height = 1 block, the peer set was empty
	// size := snap.VerSet.Size()
	// if size == 0 {
	// 	s.log.Panic("verifier set is empty!")
	// }
	// for i := uint64(0); i < uint64(size); i++ {
	// 	ver := snap.VerSet.GetVerByIndex(i)
	// 	s.log.Error("\n\n\n\ncheck snap verset first: %dth verifier %s\n\n\n", i, ver)
	// }

	_, v := snap.VerSet.GetVerByAddress(s.address)

	if v == nil {
		s.log.Error("server address is NOT in verifers set")
		return nil, errUnauthorized
	} else {
		s.log.Info("server address %s is in verifiers set", s.address)
	}

	parent := chain.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return nil, consensus.ErrBlockInvalidParentHash
	}
	// s.log.Info("[4-2-3]newBlock SealResult parent %+v", parent)

	// s.log.Info("[4-2-4]newBlock SealResult updateBlock before %+v", block)
	//update block with signature and timestamp
	block, err = s.updateBlock(parent, block) //

	if err != nil {
		s.log.Error("update block failed with err %+v", err)
		return nil, err
	}

	// wait for the timestamp of header, use this to adjust the block period
	delay := time.Unix(block.Header.CreateTimestamp.Int64(), 0).Sub(now())
	select {
	case <-time.After(delay): // wait for delay
	case <-stop: // stop is signaled
		return nil, nil
	}

	// get the proposed block hash and clear it if the seal() is completed.
	s.sealMu.Lock()
	s.proposedBlockHash = block.Hash()
	s.log.Info("assign the block hash %s to proposedBlockHash", block.Hash())
	clear := func() {
		s.proposedBlockHash = common.Hash{}
		s.sealMu.Unlock()
	}
	defer clear()

	/*
		!!! there is no commit block into commitCh, so result <- server committed channel there is no result
	*/

	// post block into Bft engine
	go s.EventMux().Post(bft.RequestEvent{
		Proposal: block,
	})

out:
	for {
		select {
		case result := <-s.commitCh:
			s.log.Info("commit channel to result %+v", result)
			// for {
			if result == nil {
				s.log.Warn("commitCh is empty")
				break
				// time.Sleep(1 * time.Second)
				// goto out
			}
			// if the block hash and the hash from channel are the same,
			// return the result. Otherwise, keep waiting the next hash.
			// MORE TEST Here (ensure logic is right here)
			if block.Hash() == result.Hash() {
				s.log.Info("get result back %s height %d", block.Hash(), block.Height())
				return result, nil
			}
			// }
		case <-stop:
			s.log.Info("commit chanel get stop signal")
			break out
			// default:
			// 	s.log.Error("shoule never reach here")
			// 	return nil, errors.New("select enter into default, namely no result and no stop signal")
		}

	}
	return nil, nil
}

// verifyHeader !!!
// verify 1.consensus- 2.createTime- 3.extraData- 4.the block is not voting on add or remove one verifier-difficulty
func (s *server) verifyHeader(chain consensus.ChainReader, header *types.BlockHeader, parents []*types.BlockHeader) error {
	err := s.verifyHeaderCommon(header, parents)
	if err != nil {
		return err
	}
	return s.verifyBFTCore(chain, header, parents)
}

// verifyHeaderCommon verify some fields of Header
func (s *server) verifyHeaderCommon(header *types.BlockHeader, parents []*types.BlockHeader) error {
	if header.Consensus != types.BftConsensus {
		fmt.Printf("verifyHeaderCommon[185] header.Consensus (%d) != types.BftConsensus (%d)\n", header.Consensus, types.BftConsensus)
		return errBFTConsensus
	}
	if header.CreateTimestamp.Cmp(big.NewInt(now().Unix())) > 0 {
		return consensus.ErrBlockCreateTimeOld
	}
	if _, err := types.ExtractBftExtra(header); err != nil {
		return errExtraDataFormatInvalid
	}
	if header.Height != 0 && !bytes.Equal(header.Witness[:], nonceAuthVote) && !bytes.Equal(header.Witness[:], nonceDropVote) {
		return errNonceInvalid
	}
	if header.Difficulty == nil || header.Difficulty.Cmp(defaultDifficulty) != 0 {
		if header.Difficulty == nil {
			s.log.Error("header.Difficulty is empty")
		}
		if header.Difficulty.Cmp(defaultDifficulty) != 0 {
			s.log.Error("header.Difficulty %d is not deafultDifficulty %d", header.Difficulty, defaultDifficulty)
		}
		return errDifficultyInvalid
	}
	return nil
}

// verifyBFTCore verify BFT consectiveness, signatures and committed seeles
func (s *server) verifyBFTCore(chain consensus.ChainReader, header *types.BlockHeader, parents []*types.BlockHeader) error {
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
		return errTimestampInvalid
	}
	// verify extraData. Verifiers in snapshot and extraData should be the same
	// s.log.Error("verfify BFTCore, snapshot")
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
	// check height, if 0 (genesis) return nil
	number := header.Height
	if number == 0 {
		return nil
	}
	// get snapshot of previous height
	snap, err := s.snapshot(chain, number-1, header.PreviousBlockHash, parents)
	if err != nil {
		return err
	}
	// get extra data
	extra, err := types.ExtractBftExtra(header)
	if err != nil {
		return err
	}
	// if extra is empty, return error
	if len(extra.CommittedSeal) == 0 {
		return errEmptyCommittedSeals
	}
	verifiers := snap.VerSet.Copy()
	validSealCount := 0
	proposalSeal := bftCore.PrepareCommittedSeal(header.Hash())
	// 1. get committed seals from current header
	for _, seal := range extra.CommittedSeal {
		addr, err := bft.GetSignatureAddress(proposalSeal, seal)
		if err != nil {
			s.log.Error("not a valid address")
			return errInvalidSignature
		}
		if verifiers.RemoveVerifier(addr) {
			validSealCount++
		} else {
			return errCommittedSealsInvalid
		}
	}
	// 2. The length of validSeal should be larger than number of faulty node + 1
	// if validSealCount <= 2*snap.VerSet.F() { // FIXME <= or <??
	if validSealCount < 2*snap.VerSet.F() {
		fmt.Println("validSealCount ", validSealCount, "require ", 2*snap.VerSet.F())
		return errCommittedSealsInvalid
	}
	return nil
}

func (s *server) verifySigner(chain consensus.ChainReader, header *types.BlockHeader, parents []*types.BlockHeader) error {
	number := header.Height
	if number == 0 {
		return errBlockUnknown
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
	if _, v := snap.VerSet.GetVerByAddress(signer); v == nil {
		return errUnauthorized
	}
	return nil
}

// VerifySeal : make sure the signers are in parent's verifier set
func (s *server) VerifySeal(chain consensus.ChainReader, header *types.BlockHeader) error {
	height := header.Height
	if height == 0 { //
		fmt.Printf("height = %+v\n", height)
		return errBlockUnknown
	}
	if header.Difficulty.Cmp(defaultDifficulty) != 0 {
		return errDifficultyInvalid
	}
	return s.verifySigner(chain, header, nil)
}

// Author retrieves the Seele address of the account that minted the given
// block, which may be different from the header's coinbase if a consensus
// engine is based on signatures.
func (s *server) Creator(header *types.BlockHeader) (common.Address, error) {
	return extractAccount(header)
}

// extractAccount extracts the account address from a signed header.
func extractAccount(header *types.BlockHeader) (common.Address, error) {
	hash := header.Hash()
	if addr, ok := cachedAddrs.Get(hash); ok {
		return addr.(common.Address), nil
	}
	bftExtra, err := types.ExtractBftExtra(header) //
	if err != nil {
		return common.Address{}, err
	}
	addr, err := bft.GetSignatureAddress(sigHash(header).Bytes(), bftExtra.Seal)
	if err != nil {
		return addr, err
	}
	cachedAddrs.Add(hash, addr)
	return addr, nil
}

// FIXME: Need to update this for bft
// sigHash returns the hash which is used as input for the Bft
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
// sigHash FIXME : here we use IstanbulFilteredHeader method, should we keep it or implement otherway?

func sigHash(header *types.BlockHeader) (hash common.Hash) {
	h := types.IstanbulFilteredHeader(header, false) //TODO
	return crypto.MustHash(h)
}

// snapshot retrieves the authorization snapshot at a given point in time.
// snapshot used to verfify the authentication.
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
			ser.log.Info("at height: %d, got snap from the RAM %+v", height, snap)
			ser.log.Info("at height: %d, verset %+v", height, snap.VerSet.GetVerByIndex(0))
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if height%checkInterval == 0 {
			if s, err := retrieveSnapshot(ser.config.Epoch, ser.db, hash); err == nil {
				ser.log.Info("Loaded voting snapshot form disk. height: %d. hash %s", height, hash)
				snap = s
				break
			}
		}

		// If we're at block zero, make a snapshot
		if height == 0 {
			genesis := chain.GetHeaderByHeight(0)
			// we do to initiate the genesis block right, otherwise verifyHeader can not pass.
			if err := ser.VerifyHeader(chain, genesis); err != nil {
				fmt.Println("failed to verify header when [snapshot] with err", err)
				return nil, err
			}
			bftExtra, err := types.ExtractBftExtra(genesis)
			if err != nil {
				return nil, err
			}
			snap = newSnapshot(ser.config.Epoch, 0, genesis.Hash(), verifier.NewVerifierSet(bftExtra.Verifiers, ser.config.ProposerPolicy))
			// FIXME need to save not so frequently and save to ser.recents
			if err := snap.save(ser.db); err != nil {
				return nil, err
			}
			ser.log.Info("Stored genesis voting snapshot to disk")
			break
		} else {
			h := chain.GetHeaderByHeight(height)
			// we do to initiate the genesis block right, otherwise verifyHeader can not pass.
			if err := ser.VerifyHeader(chain, h); err != nil {
				fmt.Println("failed to verify header when [snapshot] with err", err)
				return nil, err
			}
			bftExtra, err := types.ExtractBftExtra(h)
			if err != nil {
				return nil, err
			}
			snap = newSnapshot(ser.config.Epoch, height, h.Hash(), verifier.NewVerifierSet(bftExtra.Verifiers, ser.config.ProposerPolicy))
			// FIXME need to save not so frequently and save to ser.recents
			if err := snap.save(ser.db); err != nil {
				return nil, err
			}
			ser.log.Info("Stored genesis voting snapshot to disk")
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
	ser.log.Info("before applying headers, snapshot %+v", snap)
	snap, err := snap.applyHeaders(headers)
	if err != nil {
		return nil, err
	}
	ser.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Height%checkInterval == 0 && len(headers) > 0 {
		if err = snap.save(ser.db); err != nil {
			return nil, err
		}
		ser.log.Info("Stored voting snapshot to disk. height %d. hash %s", snap.Height, snap.Hash)
	}
	ser.log.Info("take a snapshot %+v with err %+v", snap, err)
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

// updateBlock update timestamp and signature of the block based on its number of transactions
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
		return errCommittedSealsInvalid
	}

	for _, seal := range committedSeals {
		if len(seal) != types.BftExtraSeal {
			return errCommittedSealsInvalid
		}
	}

	bftExtra, err := types.ExtractBftExtra(h)
	if err != nil {
		return err
	}

	bftExtra.CommittedSeal = make([][]byte, len(committedSeals))
	copy(bftExtra.CommittedSeal, committedSeals)

	payload, err := rlp.EncodeToBytes(&bftExtra)
	if err != nil {
		return err
	}

	h.ExtraData = append(h.ExtraData[:types.BftExtraVanity], payload...)
	return nil
}
