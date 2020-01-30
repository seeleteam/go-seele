/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package api

import (
	"errors"

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

// GetTxPoolContent returns the transactions contained within the transaction pool
func (api *TransactionPoolAPI) GetTxPoolContent() (map[string][]map[string]interface{}, error) {
	txPool := api.s.TxPoolBackend()
	data := txPool.GetTransactions(true, true)

	content := make(map[string][]map[string]interface{})
	for _, tx := range data {
		key := tx.Data.From.Hex()
		content[key] = append(content[key], PrintableOutputTx(tx))
	}

	return content, nil
}

// GetTxPoolTxCount returns the number of transaction in the pool
func (api *TransactionPoolAPI) GetTxPoolTxCount() (uint64, error) {
	txPool := api.s.TxPoolBackend()
	return uint64(txPool.GetTxCount()), nil
}

// GetPendingTransactions returns all pending transactions
func (api *TransactionPoolAPI) GetPendingTransactions() ([]map[string]interface{}, error) {
	pendingTxs := api.s.TxPoolBackend().GetTransactions(false, true)
	transactions := make([]map[string]interface{}, 0)
	for _, tx := range pendingTxs {
		transactions = append(transactions, PrintableOutputTx(tx))
	}

	return transactions, nil
}
