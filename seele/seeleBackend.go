package seele

import (
	"math/big"

	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

type SeeleBackend struct {
	s *SeeleService
}

func NewSeeleBackend(s *SeeleService) *SeeleBackend {
	return &SeeleBackend{s}
}

func (sd *SeeleBackend) TxPoolBackend() api.Pool { return sd.s.txPool }

func (sd *SeeleBackend) GetNetVersion() uint64 { return sd.s.networkID }

func (sd *SeeleBackend) GetP2pServer() *p2p.Server { return sd.s.p2pServer }

func (sd *SeeleBackend) ChainBackend() api.Chain { return sd.s.chain }

func (sd *SeeleBackend) Log() *log.SeeleLog { return sd.s.log }

func (sd *SeeleBackend) ProtocolBackend() api.Protocol { return sd.s.seeleProtocol }

// GetBlockByHeightOrHash returns the requested block.
func (sd *SeeleBackend) GetBlock(hash common.Hash, height int64) (*types.Block, error) {
	var block *types.Block
	var err error
	if hash.IsEmpty() {
		store := sd.s.chain.GetStore()
		block, err = store.GetBlock(hash)
		if err != nil {
			return nil, err
		}
	} else {
		if height < 0 {
			header := sd.s.chain.CurrentHeader()
			block, err = sd.s.chain.GetStore().GetBlockByHeight(header.Height)
		} else {
			block, err = sd.s.chain.GetStore().GetBlockByHeight(uint64(height))
		}
		if err != nil {
			return nil, err
		}
	}

	return block, nil
}

func (sd *SeeleBackend) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	store := sd.s.chain.GetStore()
	return store.GetBlockTotalDifficulty(hash)
}
