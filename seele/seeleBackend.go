package seele

import (
	"math/big"

	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
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

func (sd *SeeleBackend) GetNetVersion() string { return sd.s.netVersion }

func (sd *SeeleBackend) GetNetWorkID() string { return sd.s.networkID }

func (sd *SeeleBackend) GetP2pServer() *p2p.Server { return sd.s.p2pServer }

func (sd *SeeleBackend) ChainBackend() api.Chain { return sd.s.chain }

func (sd *SeeleBackend) Log() *log.SeeleLog { return sd.s.log }

func (sd *SeeleBackend) ProtocolBackend() api.Protocol { return sd.s.seeleProtocol }

// GetBlock returns the requested block by hash or height
func (sd *SeeleBackend) GetBlock(hash common.Hash, height int64) (*types.Block, error) {
	var block *types.Block
	var err error
	if !hash.IsEmpty() {
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

// GetReceiptByTxHash get receipt by transaction hash
func (sd *SeeleBackend) GetReceiptByTxHash(hash common.Hash) (*types.Receipt, error) {
	store := sd.s.chain.GetStore()
	receipt, err := store.GetReceiptByTxHash(hash)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (sd *SeeleBackend) GetTransaction(pool api.PoolCore, bcStore store.BlockchainStore, txHash common.Hash) (*types.Transaction, *api.BlockIndex, error) {
	return api.GetTransaction(pool, bcStore, txHash)
}
