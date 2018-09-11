package light

import (
	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

func (s *ServiceClient) TxPoolInterface() api.Pool { return s.txPool }

func (s *ServiceClient) DebtPool() *core.DebtPool { return s.debtPool }

func (s *ServiceClient) GetProtocolVersion() (uint, error) {
	return s.seeleProtocol.Protocol.Version, nil
}

func (s *ServiceClient) NetVersion() uint64 { return s.networkID }

func (s *ServiceClient) GetP2pServer() *p2p.Server { return s.p2pServer }

func (s *ServiceClient) Chain() api.Chain { return s.chain }

//@todo
func (s *ServiceClient) IsMining() bool { return false }

//@todo
func (s *ServiceClient) GetThreads() int { return 0 }

func (s *ServiceClient) Log() *log.SeeleLog { return s.log }

//@todo
func (s *ServiceClient) GetMinerCoinbase() common.Address { return common.EmptyAddress }
