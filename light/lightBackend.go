package light

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
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

// ProtocolBackend gets instance of seeleProtocol
func (l *LightBackend) ProtocolBackend() api.Protocol { return l.s.seeleProtocol }

// GetBlock gets a specific block through block's hash and height
func (l *LightBackend) GetBlock(hash common.Hash, height int64) (*types.Block, error) {
	request := &odrBlock{Hash: hash}

	if hash.IsEmpty() {
		if height < 0 {
			request.Height = l.ChainBackend().CurrentHeader().Height
		} else {
			request.Height = uint64(height)
		}
	}

	response, err := l.s.odrBackend.retrieve(request)
	if err != nil {
		return nil, err
	}

	return response.(*odrBlock).Block, nil
}

// GetBlockTotalDifficulty gets total difficulty by block hash
func (l *LightBackend) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	return l.ChainBackend().GetStore().GetBlockTotalDifficulty(hash)
}

// GetReceiptByTxHash gets block's receipt by block hash
func (l *LightBackend) GetReceiptByTxHash(hash common.Hash) (*types.Receipt, error) {
	response, err := l.s.odrBackend.retrieve(&odrReceiptRequest{TxHash: hash})
	if err != nil {
		return nil, err
	}
	result := response.(*odrReceiptResponse)
	return result.Receipt, nil
}

// GetTransaction gets tx, block index and its debt by tx hash
func (l *LightBackend) GetTransaction(pool api.PoolCore, bcStore store.BlockchainStore, txHash common.Hash) (*types.Transaction, *api.BlockIndex, error) {
	response, err := l.s.odrBackend.retrieve(&odrTxByHashRequest{TxHash: txHash})
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
