package server

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/bft"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/rpc"
)

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

// Start implements consensus.Istanbul.Start
func (s *server) Start(chain consensus.ChainReader, currentBlock func() *types.Block, hasBadBlock func(hash common.Hash) bool) error {
	s.coreMu.Lock()
	defer s.coreMu.Unlock()
	// check engine status, if already started, just return error.
	if s.coreStarted {
		return bft.ErrStartedEngine
	}

	// clear previous data
	s.proposedBlockHash = common.Hash{}
	if s.commitCh != nil {
		close(s.commitCh)
	}
	s.commitCh = make(chan *types.Block, 1)

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
	s.coreMu.Lock()
	defer s.coreMu.Unlock()
	if !s.coreStarted {
		return bft.ErrStoppedEngine
	}
	if err := s.core.Stop(); err != nil {
		return err
	}
	s.coreStarted = false
	return nil
}
