/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

// PrivateDebugAPI provides an API to access full node-related information for debugging.
type PrivateDebugAPI struct {
	s *ServiceClient
}

// NewPrivateDebugAPI creates a new NewPrivateDebugAPI object for rpc service.
func NewPrivateDebugAPI(s *ServiceClient) *PrivateDebugAPI {
	return &PrivateDebugAPI{s}
}

// PrintBlock retrieves a block and returns its pretty printed form, when height is negative the chain head is returned
func (api *PrivateDebugAPI) PrintBlock(height int64) (*types.Block, error) {
	block, err := getBlock(api.s.chain, height)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// GetTxPoolContent returns the transactions contained within the transaction pool
func (api *PrivateDebugAPI) GetTxPoolContent() (map[string][]map[string]interface{}, error) {
	txPool := api.s.TxPool()
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
	txPool := api.s.TxPool()
	return uint64(txPool.GetPendingTxCount()), nil
}

// GetPendingTransactions returns all pending transactions
func (api *PrivateDebugAPI) GetPendingTransactions() ([]map[string]interface{}, error) {
	pendingTxs := api.s.TxPool().GetTransactions(true, true)
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

// DumpHeap dumps the heap usage.
func (api *PrivateDebugAPI) DumpHeap(fileName string, gcBeforeDump bool) (string, error) {
	if len(fileName) == 0 {
		fileName = "heap.dump"
	}

	if gcBeforeDump {
		runtime.GC()
	}

	flie := filepath.Join(common.GetDefaultDataFolder(), fileName)
	f, err := os.Create(flie)
	if err != nil {
		return "", err
	}

	return flie, pprof.WriteHeapProfile(f)
}
