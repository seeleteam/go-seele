package light

import (
	"math/big"

	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

var errTransactionVerifyFailed = errors.New("got a transaction, but verify it failed")
var errReceiptVerifyFailed = errors.New("got a receipt, but verify it failed")
var errReceipIndexNil = errors.New("got a nil receipt index")

// LightBackend represents a channel (client) that communicate with backend node service.
type LightBackend struct {
	s *ServiceClient
}

// NewLightBackend creates a LightBackend
func NewLightBackend(s *ServiceClient) *LightBackend {
	return &LightBackend{s}
}

// TxPoolBackend gets the instance of tx pool
func (l *LightBackend) TxPoolBackend() api.Pool { return l.s.txPool }

// GetNetVersion gets the network version
func (l *LightBackend) GetNetVersion() string { return l.s.netVersion }

// GetNetWorkID gets the network id
func (l *LightBackend) GetNetWorkID() string { return l.s.networkID }

// GetP2pServer gets instance of p2pServer
func (l *LightBackend) GetP2pServer() *p2p.Server { return l.s.p2pServer }

// ChainBackend gets instance of blockchain
func (l *LightBackend) ChainBackend() api.Chain { return l.s.chain }

// Log gets instance of log
func (l *LightBackend) Log() *log.SeeleLog { return l.s.log }

func (l *LightBackend) IsSyncing() bool {
	return l.s.seeleProtocol.downloader.syncStatus == statusDownloading
}

// ProtocolBackend gets instance of seeleProtocol
func (l *LightBackend) ProtocolBackend() api.Protocol { return l.s.seeleProtocol }

// GetBlock gets a specific block through block's hash and height
func (l *LightBackend) GetBlock(hash common.Hash, height int64) (*types.Block, error) {
	request := &odrBlock{Hash: hash}
	var err error

	if hash.IsEmpty() {
		if height < 0 {
			request.Hash = l.ChainBackend().CurrentHeader().Hash()
		} else if request.Hash, err = l.ChainBackend().GetStore().GetBlockHash(uint64(height)); err != nil {
			return nil, errors.NewStackedErrorf(err, "failed to get block hash by height %v", height)
		}
	}

	filter := peerFilter{blockHash: hash}
	response, err := l.s.odrBackend.retrieveWithFilter(request, filter)
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to retrieve ODR block")
	}

	return response.(*odrBlock).Block, nil
}

// GetBlockTotalDifficulty gets total difficulty by block hash
func (l *LightBackend) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	return l.ChainBackend().GetStore().GetBlockTotalDifficulty(hash)
}

// GetReceiptByTxHash gets block's receipt by tx hash
func (l *LightBackend) GetReceiptByTxHash(hash common.Hash) (*types.Receipt, error) {
	blockHash := l.s.txPool.GetBlockHash(hash)

	filter := peerFilter{blockHash: blockHash}
	response, err := l.s.odrBackend.retrieveWithFilter(&odrReceiptRequest{TxHash: hash}, filter)

	if err != nil {
		return nil, err
	}
	result := response.(*odrReceiptResponse)
	return result.Receipt, nil
}

// GetTransaction gets tx, block index and its debt by tx hash
func (l *LightBackend) GetTransaction(pool api.PoolCore, bcStore store.BlockchainStore, txHash common.Hash) (*types.Transaction, *api.BlockIndex, error) {
	if tx := l.s.txPool.GetTransaction(txHash); tx != nil {
		return tx, nil, nil
	}

	blockHash := l.s.txPool.GetBlockHash(txHash)

	filter := peerFilter{blockHash: blockHash}
	response, err := l.s.odrBackend.retrieveWithFilter(&odrTxByHashRequest{TxHash: txHash}, filter)

	if err != nil {
		return nil, nil, err
	}

	result := response.(*odrTxByHashResponse)

	return result.Tx, result.BlockIndex, nil
}

// RemoveTransaction removes tx of the specified tx hash from tx pool.
func (l *LightBackend) RemoveTransaction(txHash common.Hash) {
	l.s.txPool.Remove(txHash)
}

// GetDebt returns the debt and its index for the specified debt hash.
func (l *LightBackend) GetDebt(debtHash common.Hash) (*types.Debt, *api.BlockIndex, error) {
	response, err := l.s.odrBackend.retrieve(&odrDebtRequest{DebtHash: debtHash})
	if err != nil {
		return nil, nil, err
	}

	result := response.(*odrDebtResponse)

	return result.Debt, result.BlockIndex, nil
}
