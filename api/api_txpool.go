/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package api

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
)

// Error variables
var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrDebtNotFound        = errors.New("debt not found")
)

// TransactionPoolAPI provides an API to access transaction pool information.
type TransactionPoolAPI struct {
	s Backend
}

// NewTransactionPoolAPI creates a new PrivateTransactionPoolAPI object for transaction pool rpc service.
func NewTransactionPoolAPI(s Backend) *TransactionPoolAPI {
	return &TransactionPoolAPI{s}
}

// PrintableReceipt converts the given Receipt to the RPC output
func PrintableReceipt(re *types.Receipt) (map[string]interface{}, error) {
	result := ""
	if re.Failed {
		result = string(re.Result)
	} else {
		result = hexutil.BytesToHex(re.Result)
	}
	outMap := map[string]interface{}{
		"result":    result,
		"poststate": re.PostState.Hex(),
		"txhash":    re.TxHash.Hex(),
		"contract":  "0x",
		"failed":    re.Failed,
		"usedGas":   re.UsedGas,
		"totalFee":  re.TotalFee,
	}

	if len(re.ContractAddress) > 0 {
		contractAddr, err := common.NewAddress(re.ContractAddress)
		if err != nil {
			return nil, err
		}

		outMap["contract"] = contractAddr.Hex()
	}

	if len(re.Logs) > 0 {
		outMap["logs"] = re.Logs
	}

	return outMap, nil
}

// GetBlockTransactionCount returns the count of transactions in the block with the given block hash or height.
func (api *TransactionPoolAPI) GetBlockTransactionCount(blockHash string, height int64) (int, error) {
	if len(blockHash) > 0 {
		return api.GetBlockTransactionCountByHash(blockHash)
	}

	return api.GetBlockTransactionCountByHeight(height)
}

// GetBlockDebtCount returns the count of debts in the block with the given block hash or height.
func (api *TransactionPoolAPI) GetBlockDebtCount(blockHash string, height int64) (int, error) {
	if len(blockHash) > 0 {
		return api.GetBlockDebtCountByHash(blockHash)
	}

	return api.GetBlockDebtCountByHeight(height)
}

// GetBlockTransactionCountByHeight returns the count of transactions in the block with the given height.
func (api *TransactionPoolAPI) GetBlockTransactionCountByHeight(height int64) (int, error) {
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return 0, err
	}

	return len(block.Transactions), nil
}

// GetBlockTransactionCountByHash returns the count of transactions in the block with the given hash.
func (api *TransactionPoolAPI) GetBlockTransactionCountByHash(blockHash string) (int, error) {
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return 0, err
	}

	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return 0, err
	}

	return len(block.Transactions), nil
}

// GetBlockDebtCountByHeight returns the count of debts in the block with the given height.
func (api *TransactionPoolAPI) GetBlockDebtCountByHeight(height int64) (int, error) {
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return 0, err
	}

	return len(block.Debts), nil
}

// GetBlockDebtCountByHash returns the count of debts in the block with the given hash.
func (api *TransactionPoolAPI) GetBlockDebtCountByHash(blockHash string) (int, error) {
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return 0, err
	}

	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return 0, err
	}

	return len(block.Debts), nil
}

// GetTransactionByBlockIndex returns the transaction in the block with the given block hash/height and index.
func (api *TransactionPoolAPI) GetTransactionByBlockIndex(hashHex string, height int64, index uint) (map[string]interface{}, error) {
	if len(hashHex) > 0 {
		return api.GetTransactionByBlockHashAndIndex(hashHex, index)
	}

	return api.GetTransactionByBlockHeightAndIndex(height, index)
}

//GetTransactionsFrom get transaction from one account at specific height or blockhash
func (api *TransactionPoolAPI) GetTransactionsFrom(account common.Address, blockHash string, height int64) (result []map[string]interface{}, err error) {
	if len(blockHash) > 0 {
		return api.GetTransactionsFromByHash(account, blockHash)
	}
	return api.GetTransactionsFromByHeight(account, height)
}

//GetTransactionsTo get transaction to one account at specific height or blockhash
func (api *TransactionPoolAPI) GetTransactionsTo(account common.Address, blockHash string, height int64) (result []map[string]interface{}, err error) {
	if len(blockHash) > 0 {
		return api.GetTransactionsToByHash(account, blockHash)
	}
	return api.GetTransactionsToByHeight(account, height)
}

// GetTransactionsFromByHash get transaction from one account at specific blockhash
func (api *TransactionPoolAPI) GetTransactionsFromByHash(account common.Address, blockHash string) (result []map[string]interface{}, err error) {
	var txCount = 0
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return nil, err
	}
	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for _, tx := range txs {
		if tx.FromAccount() == account {
			txCount++
			output := map[string]interface{}{
				"transaction" + fmt.Sprintf(" %d", txCount): PrintableOutputTx(tx),
			}
			result = append(result, output)
		}
	}

	return result, nil
}

// GetTransactionsToByHash get transaction from one account at specific blockhash
func (api *TransactionPoolAPI) GetTransactionsToByHash(account common.Address, blockHash string) (result []map[string]interface{}, err error) {
	var txCount = 0
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return nil, err
	}
	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for _, tx := range txs {
		if tx.ToAccount() == account {
			txCount++
			output := map[string]interface{}{
				"transaction" + fmt.Sprintf(" %d", txCount): PrintableOutputTx(tx),
			}
			result = append(result, output)
		}
	}

	return result, nil
}

// GetTransactionsFromByHeight get transaction from one account at specific height
func (api *TransactionPoolAPI) GetTransactionsFromByHeight(account common.Address, height int64) (result []map[string]interface{}, err error) {
	var txCount = 0
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for _, tx := range txs {
		if tx.FromAccount() == account {
			txCount++
			output := map[string]interface{}{
				"transaction" + fmt.Sprintf(" %d", txCount): PrintableOutputTx(tx),
			}
			result = append(result, output)

		}
	}
	return result, nil
}

// GetTransactionsToByHeight get transaction from one account at specific height
func (api *TransactionPoolAPI) GetTransactionsToByHeight(account common.Address, height int64) (result []map[string]interface{}, err error) {
	var txCount = 0
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for _, tx := range txs {
		if tx.ToAccount() == account {
			txCount++
			output := map[string]interface{}{
				"transaction" + fmt.Sprintf(" %d", txCount): PrintableOutputTx(tx),
			}
			result = append(result, output)
		}
	}

	return result, nil
}

// GetAccountTransactions get transaction of one account at specific height or blockhash
func (api *TransactionPoolAPI) GetAccountTransactions(account common.Address, blockHash string, height int64) (result []map[string]interface{}, err error) {
	if len(blockHash) > 0 {
		return api.GetAccountTransactionsByHash(account, blockHash)
	}
	return api.GetAccountTransactionsByHeight(account, height)
}

// GetAccountTransactionsByHash get transaction of one account at specific height
func (api *TransactionPoolAPI) GetAccountTransactionsByHash(account common.Address, blockHash string) (result []map[string]interface{}, err error) {
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return nil, err
	}
	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for i, tx := range txs {
		if tx.FromAccount() == account || tx.ToAccount() == account {
			output := map[string]interface{}{
				"transaction" + fmt.Sprintf(" %d", i): PrintableOutputTx(tx),
			}
			result = append(result, output)
		}
	}
	return result, nil
}

// GetAccountTransactionsByHeight get transaction of one account at specific blockhash
func (api *TransactionPoolAPI) GetAccountTransactionsByHeight(account common.Address, height int64) (result []map[string]interface{}, err error) {
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for i, tx := range txs {
		if tx.FromAccount() == account || tx.ToAccount() == account {
			output := map[string]interface{}{
				"transaction" + fmt.Sprintf(" %d", i): PrintableOutputTx(tx),
			}
			result = append(result, output)
		}
	}
	return result, nil
}

// GetBlockTransactions get all txs in the block with heigth or blockhash
func (api *TransactionPoolAPI) GetBlockTransactions(blockHash string, height int64) (result []map[string]interface{}, err error) {
	if len(blockHash) > 0 {
		return api.GetBlockTransactionsByHash(blockHash)
	}

	return api.GetBlockTransactionsByHeight(height)
}

// GetBlockTransactionsByHeight returns the transactions in the block with the given height.
func (api *TransactionPoolAPI) GetBlockTransactionsByHeight(height int64) (result []map[string]interface{}, err error) {
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for i, tx := range txs {
		output := map[string]interface{}{
			"transaction" + fmt.Sprintf(" %d", i+1): PrintableOutputTx(tx),
		}
		result = append(result, output)
	}
	return result, nil
}

// GetBlockTransactionsByHash returns the transactions in the block with the given height.
func (api *TransactionPoolAPI) GetBlockTransactionsByHash(blockHash string) (result []map[string]interface{}, err error) {
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return nil, err
	}

	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for i, tx := range txs {
		output := map[string]interface{}{
			"transaction" + fmt.Sprintf(" %d", i+1): PrintableOutputTx(tx),
		}
		result = append(result, output)
	}
	return result, nil
}

// GetTransactionByBlockHeightAndIndex returns the transaction in the block with the given block height and index.
func (api *TransactionPoolAPI) GetTransactionByBlockHeightAndIndex(height int64, index uint) (map[string]interface{}, error) {
	block, err := api.s.GetBlock(common.EmptyHash, height)
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
func (api *TransactionPoolAPI) GetTransactionByBlockHashAndIndex(hashHex string, index uint) (map[string]interface{}, error) {
	hash, err := common.HexToHash(hashHex)
	if err != nil {
		return nil, err
	}

	block, err := api.s.GetBlock(hash, 0)
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
func (api *TransactionPoolAPI) GetReceiptByTxHash(txHash, abiJSON string) (map[string]interface{}, error) {
	hash, err := common.HexToHash(txHash)
	if err != nil {
		return nil, err
	}

	receipt, err := api.s.GetReceiptByTxHash(hash)
	if err != nil {
		return nil, err
	}

	return printReceiptByABI(api, receipt, abiJSON)
}

// GetReceiptsByBlockHash get receipts by block hash
func (api *TransactionPoolAPI) GetReceiptsByBlockHash(blockHash string) (map[string]interface{}, error) {
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return nil, err
	}

	receipts, err := api.s.ChainBackend().GetStore().GetReceiptsByBlockHash(hash)
	if err != nil {
		return nil, err
	}

	outMaps := make([]map[string]interface{}, 0, len(receipts))
	for _, re := range receipts {
		outMap, err := PrintableReceipt(re)
		if err != nil {
			return nil, err
		}
		outMaps = append(outMaps, outMap)
	}

	return map[string]interface{}{
		"blockHash": blockHash,
		"receipts":  outMaps,
	}, nil
}

// GetTransactionByHash returns the transaction by the given transaction hash.
func (api *TransactionPoolAPI) GetTransactionByHash(txHash string) (map[string]interface{}, error) {
	hashByte, err := hexutil.HexToBytes(txHash)
	if err != nil {
		return nil, err
	}
	hash := common.BytesToHash(hashByte)

	tx, idx, err := api.s.GetTransaction(api.s.TxPoolBackend(), api.s.ChainBackend().GetStore(), hash)
	if err != nil {
		api.s.Log().Debug("Failed to get transaction by hash, %v", err.Error())
		return nil, err
	}

	if tx == nil {
		return nil, nil
	}

	output := map[string]interface{}{
		"transaction": PrintableOutputTx(tx),
	}

	debt := types.NewDebtWithContext(tx)
	if debt != nil {
		output["debt"] = debt
	}

	if idx == nil {
		output["status"] = "pool"
	} else {
		output["status"] = "block"

		output["blockHash"] = idx.BlockHash.Hex()
		output["blockHeight"] = idx.BlockHeight
		output["txIndex"] = idx.Index
	}

	return output, nil
}

// BlockIndex represents the index info in a block.
type BlockIndex struct {
	BlockHash   common.Hash
	BlockHeight uint64
	Index       uint
}

// GetTransaction returns the transaction by the given blockchain store and transaction hash.
func GetTransaction(pool PoolCore, bcStore store.BlockchainStore, txHash common.Hash) (*types.Transaction, *BlockIndex, error) {
	// Try to get transaction in tx pool.
	if tx := pool.GetTransaction(txHash); tx != nil {
		return tx, nil, nil
	}

	// Try to find transaction in blockchain.
	txIdx, err := bcStore.GetTxIndex(txHash)
	if err != nil {
		return nil, nil, err
	}

	if txIdx == nil {
		return nil, nil, nil
	}

	block, err := bcStore.GetBlock(txIdx.BlockHash)
	if err != nil {
		return nil, nil, err
	}

	tx := block.Transactions[txIdx.Index]
	idx := &BlockIndex{block.HeaderHash, block.Header.Height, txIdx.Index}
	return tx, idx, nil
}

// GetDebt returns the debt for the specified debt hash.
func GetDebt(pool *core.DebtPool, bcStore store.BlockchainStore, debtHash common.Hash) (*types.Debt, *BlockIndex, error) {
	// Try to get the debt in debt pool.
	if debt := pool.GetDebtByHash(debtHash); debt != nil {
		return debt, nil, nil
	}

	// Try to find debt in blockchain.
	debtIdx, err := bcStore.GetDebtIndex(debtHash)
	if err != nil {
		return nil, nil, err
	}

	block, err := bcStore.GetBlock(debtIdx.BlockHash)
	if err != nil {
		return nil, nil, err
	}

	debt := block.Debts[debtIdx.Index]
	idx := &BlockIndex{
		BlockHash:   block.HeaderHash,
		BlockHeight: block.Header.Height,
		Index:       debtIdx.Index,
	}

	return debt, idx, nil
}

// GetGasPrice get tx gas price
func (api *TransactionPoolAPI) GetGasPrice(txHash string) (map[string]interface{}, error) {
	hashByte, err := hexutil.HexToBytes(txHash)
	if err != nil {
		return nil, err
	}
	hash := common.BytesToHash(hashByte)
	tx, _, err := api.s.GetTransaction(api.s.TxPoolBackend(), api.s.ChainBackend().GetStore(), hash)
	// if err != nil {
	// 	api.s.Log().Info("Failed to get transaction by hash, %v", err.Error())
	// 	return nil, err
	// }
	if tx == nil || err != nil {
		resultNotFound := map[string]interface{}{
			"error": "the transaction not found",
		}
		return resultNotFound, nil
	}
	// var err error
	gasPrice := map[string]interface{}{
		"gasPrice": tx.Data.GasPrice,
	}
	return gasPrice, nil
}
