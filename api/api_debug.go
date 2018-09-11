/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package api

import (
	"github.com/seeleteam/go-seele/core/types"
)

// PrivateDebugAPI provides an API to access full node-related information for debugging.
type PrivateDebugAPI struct {
	s Backend
}

// NewPrivateDebugAPI creates a new NewPrivateDebugAPI object for rpc service.
func NewPrivateDebugAPI(s Backend) *PrivateDebugAPI {
	return &PrivateDebugAPI{s}
}

// GetTxPoolContent returns the transactions contained within the transaction pool
func (api *PrivateDebugAPI) GetTxPoolContent() (map[string][]map[string]interface{}, error) {
	txPool := api.s.TxPoolInterface()
	data := txPool.GetTransactions(false, true)

	content := make(map[string][]map[string]interface{})
	for _, tx := range data {
		key := tx.Data.From.ToHex()
		content[key] = append(content[key], PrintableOutputTx(tx))
	}

	return content, nil
}

// GetTxPoolTxCount returns the number of transaction in the pool
func (api *PrivateDebugAPI) GetTxPoolTxCount() (uint64, error) {
	txPool := api.s.TxPoolInterface()
	return uint64(txPool.GetPendingTxCount()), nil
}

// GetPendingTransactions returns all pending transactions
func (api *PrivateDebugAPI) GetPendingTransactions() ([]map[string]interface{}, error) {
	pendingTxs := api.s.TxPoolInterface().GetTransactions(true, true)
	transactions := make([]map[string]interface{}, 0)
	for _, tx := range pendingTxs {
		transactions = append(transactions, PrintableOutputTx(tx))
	}

	return transactions, nil
}

// GetPendingDebts returns all pending debts
func (api *PrivateDebugAPI) GetPendingDebts() ([]*types.Debt, error) {
	return api.s.DebtPool().GetAll(), nil
}
