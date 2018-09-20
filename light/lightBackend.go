package light

import (
	"errors"
	"fmt"
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

func (l *LightBackend) GetMinerCoinbase() common.Address { return common.EmptyAddress }

func (l *LightBackend) ProtocolBackend() api.Protocol { return l.s.seeleProtocol }

func (l *LightBackend) GetBlockByHash(hash common.Hash) (*types.Block, error) {
	if hash.IsEmpty() {
		return nil, errors.New("request hash is empty")
	}

	request := &odrBlock{Hash: hash}

	if err := l.s.odrBackend.sendRequest(request); err != nil {
		return nil, fmt.Errorf("Failed to send request to peers, %v", err.Error())
	}

	if err := request.getError(); err != nil {
		return nil, err
	}

	return request.Block, nil
}

//@todo
func (l *LightBackend) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	return nil, nil
}

func (l *LightBackend) GetBlockByHeight(height int64) (*types.Block, error) {
	if height <= 0 {
		height = 0
	}
	request := &odrBlock{Height: uint64(height)}

	if err := l.s.odrBackend.sendRequest(request); err != nil {
		return nil, fmt.Errorf("Failed to send request to peers, %v", err.Error())
	}

	if err := request.getError(); err != nil {
		return nil, err
	}

	return request.Block, nil
}
