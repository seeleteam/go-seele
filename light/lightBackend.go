package light

import (
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

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
	var request *odrBlock
	request = &odrBlock{Height: height, Hash: hash}

	if err := l.s.odrBackend.sendRequest(request); err != nil {
		return nil, fmt.Errorf("Failed to send request to peers, %v", err.Error())
	}

	if err := request.getError(); err != nil {
		return nil, err
	}

	if err := request.Validate(l.s.chain.GetStore()); err != nil {
		return nil, err
	}

	return request.Block, nil
}

// GetBlockTotalDifficulty gets total difficulty by block hash
func (l *LightBackend) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	return l.ChainBackend().GetStore().GetBlockTotalDifficulty(hash)
}

// GetReceiptByTxHash gets block's receipt by block hash
func (l *LightBackend) GetReceiptByTxHash(hash common.Hash) (*types.Receipt, error) {
	var request *odrtReceipt
	request = &odrtReceipt{TxHash: hash}

	if err := l.s.odrBackend.sendRequest(request); err != nil {
		return nil, fmt.Errorf("Failed to send request to peers, %v", err.Error())
	}

	if err := request.getError(); err != nil {
		return nil, err
	}
	return request.Receipt, nil
}

// GetTransaction gets tx, block index and its debt by tx hash
func (l *LightBackend) GetTransaction(pool api.PoolCore, bcStore store.BlockchainStore, txHash common.Hash) (*types.Transaction, *api.BlockIndex, *types.Debt, error) {
	request := &odrTxByHash{TxHash: txHash}
	if err := l.s.odrBackend.sendRequest(request); err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to send request to peers, %v", err.Error())
	}

	if err := request.getError(); err != nil {
		return nil, nil, nil, err
	}

	return request.Tx, request.BlockIndex, request.Debt, nil
}
