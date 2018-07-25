/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/types"
)

// PrivateDebugAPI provides an API to access full node-related information for debug.
type PrivateDebugAPI struct {
	s *SeeleService
}

// NewPrivateDebugAPI creates a new NewPrivateDebugAPI object for rpc service.
func NewPrivateDebugAPI(s *SeeleService) *PrivateDebugAPI {
	return &PrivateDebugAPI{s}
}

// GetBlockRlp retrieves the RLP encoded for of a single block, when height is -1 the chain head is returned
func (api *PrivateDebugAPI) GetBlockRlp(height *int64, result *string) error {
	block, err := getBlock(api.s.chain, *height)
	if err != nil {
		return err
	}

	blockRlp, err := common.Serialize(block)
	if err != nil {
		return err
	}

	*result = hexutil.BytesToHex(blockRlp)
	return nil
}

// PrintBlock retrieves a block and returns its pretty printed form, when height is -1 the chain head is returned
func (api *PrivateDebugAPI) PrintBlock(height *int64, result *types.Block) error {
	block, err := getBlock(api.s.chain, *height)
	if err != nil {
		return err
	}

	*result = *block
	return nil
}

// GetTxPoolContent returns the transactions contained within the transaction pool
func (api *PrivateDebugAPI) GetTxPoolContent(input interface{}, result *map[string][]map[string]interface{}) error {
	txPool := api.s.TxPool()
	data := txPool.GetTransactions(false, true)

	content := make(map[string][]map[string]interface{})
	for _, tx := range data {
		key := tx.Data.From.ToHex()
		content[key] = append(content[key], PrintableOutputTx(tx))
	}
	*result = content

	return nil
}

// GetTxPoolTxCount returns the number of transaction in the pool
func (api *PrivateDebugAPI) GetTxPoolTxCount(input interface{}, result *uint64) error {
	txPool := api.s.TxPool()
	*result = uint64(txPool.GetPendingTxCount())
	return nil
}

// TpsInfo tps detail info
type TpsInfo struct {
	StartHeight uint64
	EndHeight   uint64
	Count       uint64
	Duration    uint64
}

// GetTPS get tps info
func (api *PrivateDebugAPI) GetTPS(input interface{}, result *TpsInfo) error {
	chain := api.s.BlockChain()
	block := chain.CurrentBlock()
	timeInterval := uint64(150)
	if block.Header.Height == 0 {
		return nil
	}

	var count = uint64(len(block.Transactions) - 1)
	var duration uint64
	var endHeight uint64
	startTime := block.Header.CreateTimestamp.Uint64()
	for height := block.Header.Height - 1; height > 0; height-- {
		current, err := chain.GetStore().GetBlockByHeight(height)
		if err != nil {
			return fmt.Errorf("failed to get block, error:%s, block height:%d", err, height)
		}

		count += uint64(len(current.Transactions) - 1)
		duration = startTime - current.Header.CreateTimestamp.Uint64()
		endHeight = height

		if duration > timeInterval {
			break
		}
	}

	*result = TpsInfo{
		StartHeight: endHeight,
		EndHeight:   block.Header.Height,
		Count:       count,
		Duration:    duration,
	}

	return nil
}

// DumpHeapRequest represents the heamp dump request.
type DumpHeapRequest struct {
	Filename     string
	GCBeforeDump bool
}

// DumpHeap dumps the heap usage.
func (api *PrivateDebugAPI) DumpHeap(request *DumpHeapRequest, result *interface{}) error {
	filename := "heap.dump"

	if len(request.Filename) > 0 {
		filename = request.Filename
	}

	if request.GCBeforeDump {
		runtime.GC()
	}

	f, err := os.Create(filepath.Join(common.GetDefaultDataFolder(), filename))
	if err != nil {
		return err
	}

	return pprof.WriteHeapProfile(f)
}
