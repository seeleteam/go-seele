package light

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/trie"
)

var errTransactionVerifyFailed = errors.New("got a transaction, but verify it failed")

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

	// @todo
	if _, err := l.s.odrBackend.sendRequest(request); err != nil {
		return nil, fmt.Errorf("Failed to send request to peers, %v", err.Error())
	}

	if err := request.getError(); err != nil {
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

	// @todo
	if _, err := l.s.odrBackend.sendRequest(request); err != nil {
		return nil, fmt.Errorf("Failed to send request to peers, %v", err.Error())
	}

	if err := request.getError(); err != nil {
		return nil, err
	}
	return request.Receipt, nil
}

// GetTransaction gets tx, block index and its debt by tx hash
func (l *LightBackend) GetTransaction(pool api.PoolCore, bcStore store.BlockchainStore, txHash common.Hash) (*types.Transaction, *api.BlockIndex, *types.Debt, error) {
	request := &odrTxByHashRequest{TxHash: txHash}
	result, err := l.s.odrBackend.sendRequest(request)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to send request to peers, %s", err)
	}

	if err := result.getError(); err != nil {
		return nil, nil, nil, err
	}

	response := result.(*odrTxByHashResponse)
	// verify transaction if it is packed in block
	if response.BlockIndex != nil {
		header, err := bcStore.GetBlockHeader(response.BlockIndex.BlockHash)
		if err != nil {
			return nil, nil, nil, err
		}

		proof := arrayToMap(response.Proof)
		value, err := trie.VerifyProof(header.TxHash, request.TxHash.Bytes(), proof)
		if err != nil {
			return nil, nil, nil, err
		}

		buff := common.SerializePanic(response.Tx)
		if !bytes.Equal(buff, value) {
			return nil, nil, nil, errTransactionVerifyFailed
		}
	}

	return response.Tx, response.BlockIndex, response.Debt, nil
}
