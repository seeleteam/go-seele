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

func (sd *SeeleBackend) GetBlockByHash(hashHex string) (*types.Block, error) {
	return sd.GetBlockByHash(hashHex)
}

func (sd *SeeleBackend) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	return sd.GetBlockTotalDifficulty(hash)
}

func (sd *SeeleBackend) GetBlockByHeight(height int64) (*types.Block, error) {
	return sd.GetBlockByHeight(height)
}
