package light

import (
	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

func (s *ServiceClient) TxPoolBackend() api.Pool { return s.txPool }

func (s *ServiceClient) GetNetVersion() uint64 { return s.networkID }

func (s *ServiceClient) GetP2pServer() *p2p.Server { return s.p2pServer }

func (s *ServiceClient) ChainBackend() api.Chain { return s.chain }

func (s *ServiceClient) Log() *log.SeeleLog { return s.log }

func (s *ServiceClient) GetMinerCoinbase() common.Address { return common.EmptyAddress }

func (s *ServiceClient) ProtocolBackend() api.Protocol { return s.seeleProtocol }
