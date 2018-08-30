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
)

var (
	errTransactionNotFound = errors.New("transaction not found")
	errDebtNotFound        = errors.New("debt not found")
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
func (api *PrivateTransactionPoolAPI) GetBlockTransactionCountByHeight(height int64) (int, error) {
	block, err := getBlock(api.s.chain, height)
	if err != nil {
		return 0, err
	}

	return len(block.Transactions), nil
}

// GetBlockTransactionCountByHash returns the count of transactions in the block with the given hash.
func (api *PrivateTransactionPoolAPI) GetBlockTransactionCountByHash(blockHash string) (int, error) {
	store := api.s.chain.GetStore()
	hashByte, err := hexutil.HexToBytes(blockHash)
	if err != nil {
		return 0, err
	}

	hash := common.BytesToHash(hashByte)
	block, err := store.GetBlock(hash)
	if err != nil {
		return 0, err
	}

	return len(block.Transactions), nil
}

// GetTransactionByBlockHeightAndIndex returns the transaction in the block with the given block height and index.
func (api *PrivateTransactionPoolAPI) GetTransactionByBlockHeightAndIndex(height int64, index uint) (map[string]interface{}, error) {
	block, err := getBlock(api.s.chain, height)
	if err != nil {
		return nil, err
	}

	txs := block.Transactions
	if index >= uint(len(txs)) {
		return nil, errors.New("index out of block transaction list range, the max index is " + strconv.Itoa(len(txs)-1))
	}

	return PrintableOutputTx(txs[index]), nil
}

// GetTransactionByBlockHashAndIndex returns the transaction in the block with the given block hash and index.
func (api *PrivateTransactionPoolAPI) GetTransactionByBlockHashAndIndex(hashHex string, index uint) (map[string]interface{}, error) {
	store := api.s.chain.GetStore()
	hashByte, err := hexutil.HexToBytes(hashHex)
	if err != nil {
		return nil, err
	}

	hash := common.BytesToHash(hashByte)
	block, err := store.GetBlock(hash)
	if err != nil {
		return nil, err
	}

	txs := block.Transactions
	if index >= uint(len(txs)) {
		return nil, errors.New("index out of block transaction list range, the max index is " + strconv.Itoa(len(txs)-1))
	}

	return PrintableOutputTx(txs[index]), nil
}

// GetReceiptByTxHash get receipt by transaction hash
func (api *PrivateTransactionPoolAPI) GetReceiptByTxHash(txHash string) (map[string]interface{}, error) {
	hashByte, err := hexutil.HexToBytes(txHash)
	if err != nil {
		return nil, err
	}
	hash := common.BytesToHash(hashByte)

	store := api.s.chain.GetStore()
	receipt, err := store.GetReceiptByTxHash(hash)
	if err != nil {
		return nil, err
	}
	return PrintableReceipt(receipt)
}

// GetTransactionByHash returns the transaction by the given transaction hash.
func (api *PrivateTransactionPoolAPI) GetTransactionByHash(txHash string) (map[string]interface{}, error) {
	store := api.s.chain.GetStore()
	hashByte, err := hexutil.HexToBytes(txHash)
	if err != nil {
		return nil, err
	}
	hash := common.BytesToHash(hashByte)

	output := make(map[string]interface{})

	// Try to get transaction in txpool
	tx := api.s.TxPool().GetTransaction(hash)
	if tx != nil {
		output["transaction"] = PrintableOutputTx(tx)
		output["status"] = "pool"

		return output, nil
	}

	// Try to get finalized transaction
	txIndex, err := store.GetTxIndex(hash)
	if err != nil {
		api.s.log.Info(err.Error())
		return nil, errTransactionNotFound
	}

	if txIndex != nil {
		block, err := store.GetBlock(txIndex.BlockHash)
		if err != nil {
			return nil, err
		}
		output["transaction"] = PrintableOutputTx(block.Transactions[txIndex.Index])
		output["status"] = "block"
		output["blockHash"] = block.HeaderHash.ToHex()
		output["blockHeight"] = block.Header.Height
		output["txIndex"] = txIndex.Index

		return output, nil
	}

	return nil, nil
}

// GetDebtByHash return the debt info by debt hash
func (api *PrivateTransactionPoolAPI) GetDebtByHash(debtHash string) (map[string]interface{}, error) {
	hashByte, err := hexutil.HexToBytes(debtHash)
	if err != nil {
		return nil, err
	}
	hash := common.BytesToHash(hashByte)

	output := make(map[string]interface{})
	debt := api.s.DebtPool().GetDebtByHash(hash)
	if debt != nil {
		output["debt"] = debt
		output["status"] = "pool"

		return output, nil
	}

	store := api.s.chain.GetStore()
	debtIndex, err := store.GetDebtIndex(hash)
	if err != nil {
		api.s.log.Info(err.Error())
		return nil, errDebtNotFound
	}

	if debtIndex != nil {
		block, err := store.GetBlock(debtIndex.BlockHash)
		if err != nil {
			return nil, err
		}

		output["debt"] = block.Debts[debtIndex.Index]
		output["status"] = "block"
		output["blockHash"] = block.HeaderHash.ToHex()
		output["blockHeight"] = block.Header.Height
		output["debtIndex"] = debtIndex

		return output, nil
	}

	return nil, nil
}

// GetPendingTransactions returns all pending transactions
func (api *PrivateTransactionPoolAPI) GetPendingTransactions() ([]map[string]interface{}, error) {
	pendingTxs := api.s.TxPool().GetTransactions(true, true)
	var transactions []map[string]interface{}
	for _, tx := range pendingTxs {
		transactions = append(transactions, PrintableOutputTx(tx))
	}

	return transactions, nil
}
