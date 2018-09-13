package seele

import (
	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

func (s *SeeleService) TxPoolBackend() api.Pool { return s.txPool }

func (s *SeeleService) TxPool() *core.TransactionPool { return s.txPool }

func (s *SeeleService) DebtPool() *core.DebtPool { return s.debtPool }

func (s *SeeleService) GetProtocolVersion() (uint, error) {
	return s.seeleProtocol.Protocol.Version, nil
}

func (s *SeeleService) GetNetVersion() uint64 { return s.networkID }

func (s *SeeleService) GetP2pServer() *p2p.Server { return s.p2pServer }

func (s *SeeleService) ChainBackend() api.Chain { return s.chain }

func (s *SeeleService) IsMining() bool { return s.miner.IsMining() }

func (s *SeeleService) GetThreads() int { return s.miner.GetThreads() }

func (s *SeeleService) Log() *log.SeeleLog { return s.log }

func (s *SeeleService) GetMinerCoinbase() common.Address { return s.miner.GetCoinbase() }

// AddTx add a tx to miner
func (s *SeeleService) AddTx(tx types.Transaction) (bool, error) {
	shard := tx.Data.From.Shard()
	var err error
	if shard != common.LocalShardNumber {
		if err = tx.ValidateWithoutState(true, false); err == nil {
			s.seeleProtocol.SendDifferentShardTx(&tx, shard)
		}
	} else {
		err = s.txPool.AddTransaction(&tx)
	}

	if err != nil {
		return false, err
	}

	return true, nil
}
