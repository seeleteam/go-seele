package server

import (
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/bft"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/rpc"
)

/*
here we mainly implement the Bft engine interface defined in consensus/consensus.go
type Bft interface {

	Engine

	// Start starts the engine
	Start(chain ChainReader, currentBlock func() *types.Block, hasBadBlock func(hash common.Hash) bool) error

	// Stop stops the engine
	Stop() error
}

type Engine interface {
	// Prepare header before generate block
	Prepare(chain ChainReader, header *types.BlockHeader) error

	// VerifyHeader verify block header
	VerifyHeader(chain ChainReader, header *types.BlockHeader) error

	// Seal generate block
	Seal(chain ChainReader, block *types.Block, stop <-chan struct{}, results chan<- *types.Block) error

	// APIs returns the RPC APIs this consensus engine provides.
	APIs(chain ChainReader) []rpc.API

	// SetThreads set miner threads
	SetThreads(thread int)
}

*/

func (s *server) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "bft",
		Version:   "1.0",
		Service:   &API{chain: chain, bft: s}, // TODO
		Public:    true,
	}}
}

func (s *server) SetThreads(thread int) {
	// do nothing
}

// Start implements consensus.Bft.Start
func (s *server) Start(chain consensus.ChainReader, currentBlock func() *types.Block, hasBadBlock func(hash common.Hash) bool) error {
	s.coreMu.Lock()
	defer s.coreMu.Unlock()
	// check engine status, if already started, just return error.
	if s.coreStarted { // FIXME, afer download the coreStarted is not chqnge
		return bft.ErrEngineStarted
	}
	// clear previous data
	s.proposedBlockHash = common.Hash{}
	if s.commitCh != nil {
		close(s.commitCh)
	}
	s.commitCh = make(chan *types.Block, 1)
	fmt.Println("make a new commit channel")

	s.chain = chain
	s.currentBlock = currentBlock
	s.hasBadBlock = hasBadBlock

	if err := s.core.Start(); err != nil {
		return err
	}

	s.coreStarted = true
	return nil
}

// Stop implements consensus.Bft.Stop
func (s *server) Stop() error {
	s.log.Info("BFT engine is stopping")
	s.coreMu.Lock()
	defer s.coreMu.Unlock()
	if !s.coreStarted {
		return bft.ErrEngineStopped
	}
	if err := s.core.Stop(); err != nil {
		return err
	}
	s.log.Info("BFT engine is stopping engine core")
	s.coreStarted = false
	s.log.Info("coreStarted == false? -> %t", s.coreStarted == false)
	return nil
}

// Prepare prepare a block
func (s *server) Prepare(chain consensus.ChainReader, header *types.BlockHeader) error {
	s.log.Info("Prepare a block")
	//1. setup some unused field
	// header.Creator = common.Address{}
	header.Creator = s.Address()
	header.Witness = make([]byte, bft.WitnessSize)
	// header.SecondWitness = make([]byte, bft.WitnessSize)
	header.Consensus = types.BftConsensus
	header.Difficulty = defaultDifficulty // for bft consensus algorithm, we just set difficulty as the default value

	// 2. copy parent extra data as the header extra data
	height := header.Height
	parent := chain.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return consensus.ErrBlockInvalidParentHash
	}
	// voting snapshot
	snap, err := s.snapshot(chain, height-1, header.PreviousBlockHash, nil)

	s.log.Debug("get [height-1] = %d snap %+v", height-1, snap)
	if err != nil {
		s.log.Error("snapshot return err <-", err)
		return err
	}

	//get valid candidate list
	s.candidatesLock.RLock()
	var addrs []common.Address
	var auths []bool
	for addr, auth := range s.candidates {
		s.log.Info("checkVote with addr: %+v, auth: %+v", addr, auth)
		if snap.checkVote(addr, auth) {
			addrs = append(addrs, addr)
			auths = append(auths, auth)
		}
	}
	s.candidatesLock.RUnlock()
	// pick one candidate randomly
	// the block creator will get the reward, here we randomly pickout peer
	// if err := s.GetVerifierFromSWExtra(header, addrs, auths); err != nil {
	// 	return err
	// }

	if len(addrs) > 0 { // this will be used to gurantee the block prepared by non-ver can not passed
		index := rand.Intn(len(addrs))
		header.Creator = addrs[index]
		if auths[index] { // if the address is authorized to vote, then put nonceAuthVote otherwise put nonceDropVote
			copy(header.Witness[:], nonceAuthVote)
		} else {
			copy(header.Witness[:], nonceDropVote)
		}
	}

	curVer := snap.verifiers()
	newVer := s.GetCurrentVerifiers(curVer, addrs, auths)
	curVer = append(curVer, newVer...)
	// add verifiers in snapshot to extraData's verifiers section
	s.log.Debug("[bft] Prepare a block extra with snap.verifiers %+v", snap.verifiers())
	s.log.Debug("[bft] Prepare a block extra with verifiers %+v", curVer)
	// extra, err := prepareExtra(header, curVer)
	extra, err := prepareExtra(header, snap.verifiers())
	if err != nil {
		fmt.Println("failed to prepare extra data")
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

func (s *server) VerifyHeader(chain consensus.ChainReader, header *types.BlockHeader) error {
	return s.verifyHeader(chain, header, nil)
}
