package light

import (
	"math/big"

	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

type LightBackend struct {
	s *ServiceClient
}

func NewLightBackend(s *ServiceClient) *LightBackend {
	return &LightBackend{s}
}

func (l *LightBackend) TxPoolBackend() api.Pool { return l.s.txPool }

func (l *LightBackend) GetNetVersion() uint64 { return l.s.networkID }

func (l *LightBackend) GetP2pServer() *p2p.Server { return l.s.p2pServer }

func (l *LightBackend) ChainBackend() api.Chain { return l.s.chain }

func (l *LightBackend) Log() *log.SeeleLog { return l.s.log }

func (l *LightBackend) ProtocolBackend() api.Protocol { return l.s.seeleProtocol }

//@todo
func (l *LightBackend) GetBlockByHash(hashHex string) (*types.Block, error) {
	return nil, nil
}

//@todo
func (l *LightBackend) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	return nil, nil
}

//@todo
func (l *LightBackend) GetBlockByHeight(height int64) (*types.Block, error) {
	return nil, nil
}
