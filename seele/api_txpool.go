/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"errors"
	"strconv"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/types"
)

var (
	errTransactionNotFound = errors.New("transaction not found")
)

// PrivateTransactionPoolAPI provides an API to access transaction pool information.
type PrivateTransactionPoolAPI struct {
	s *SeeleService
}

// NewPrivateTransactionPoolAPI creates a new PrivateTransactionPoolAPI object for transaction pool rpc service.
func NewPrivateTransactionPoolAPI(s *SeeleService) *PrivateTransactionPoolAPI {
	return &PrivateTransactionPoolAPI{s}
}

// GetBlockTransactionCountByHeight returns the count of transactions in the block with the given height.
func (api *PrivateTransactionPoolAPI) GetBlockTransactionCountByHeight(height *int64, result *int) error {
	block, err := getBlock(api.s.chain, *height)
	if err != nil {
		return err
	}
	*result = len(block.Transactions)
	return nil
}

// GetBlockTransactionCountByHash returns the count of transactions in the block with the given hash.
func (api *PrivateTransactionPoolAPI) GetBlockTransactionCountByHash(blockHash *string, result *int) error {
	store := api.s.chain.GetStore()
	hashByte, err := hexutil.HexToBytes(*blockHash)
	if err != nil {
		return err
	}

	hash := common.BytesToHash(hashByte)
	block, err := store.GetBlock(hash)
	if err != nil {
		return err
	}
	*result = len(block.Transactions)
	return nil
}

// GetTransactionByBlockHeightAndIndex returns the transaction in the block with the given block height and index.
func (api *PrivateTransactionPoolAPI) GetTransactionByBlockHeightAndIndex(request *GetTxByBlockHeightAndIndexRequest, result *map[string]interface{}) error {
	block, err := getBlock(api.s.chain, request.Height)
	if err != nil {
		return err
	}

	txs := block.Transactions
	if request.Index >= len(txs) {
		return errors.New("index out of block transaction list range, the max index is " + strconv.Itoa(len(txs)-1))
	}

	*result = PrintableOutputTx(txs[request.Index])
	return nil
}

// GetTransactionByBlockHashAndIndex returns the transaction in the block with the given block hash and index.
func (api *PrivateTransactionPoolAPI) GetTransactionByBlockHashAndIndex(request *GetTxByBlockHashAndIndexRequest, result *map[string]interface{}) error {
	store := api.s.chain.GetStore()
	hashByte, err := hexutil.HexToBytes(request.HashHex)
	if err != nil {
		return err
	}

	hash := common.BytesToHash(hashByte)
	block, err := store.GetBlock(hash)
	if err != nil {
		return err
	}

	txs := block.Transactions
	if request.Index >= len(txs) {
		return errors.New("index out of block transaction list range, the max index is " + strconv.Itoa(len(txs)-1))
	}
	*result = PrintableOutputTx(txs[request.Index])
	return nil
}

// GetReceiptByTxHash get receipt by transaction hash
func (api *PrivateTransactionPoolAPI) GetReceiptByTxHash(txHash *string, result *types.Receipt) error {
	hashByte, err := hexutil.HexToBytes(*txHash)
	if err != nil {
		return err
	}
	hash := common.BytesToHash(hashByte)

	store := api.s.chain.GetStore()
	receipt, err := store.GetReceiptByTxHash(hash)
	if err != nil {
		return err
	}
	*result = *receipt
	return nil
}

// GetTransactionByHash returns the transaction by the given transaction hash.
func (api *PrivateTransactionPoolAPI) GetTransactionByHash(txHash *string, result *map[string]interface{}) error {
	store := api.s.chain.GetStore()
	hashByte, err := hexutil.HexToBytes(*txHash)
	if err != nil {
		return err
	}
	hash := common.BytesToHash(hashByte)

	// Try to get transaction in txpool
	tx := api.s.TxPool().GetTransaction(hash)
	if tx != nil {
		*result = PrintableOutputTx(tx)
		return nil
	}

	// Try to get finalized transaction
	txIndex, err := store.GetTxIndex(hash)
	if err != nil {
		api.s.log.Info(err.Error())
		return errTransactionNotFound
	}

	if txIndex != nil {
		block, err := store.GetBlock(txIndex.BlockHash)
		if err != nil {
			return err
		}
		*result = PrintableOutputTx(block.Transactions[txIndex.Index])
		return nil
	}

	return nil
}
