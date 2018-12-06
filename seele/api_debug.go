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

// PrintBlock retrieves a block and returns its pretty printed form, when height is -1 the chain head is returned
func (api *PrivateDebugAPI) PrintBlock(height int64) (*types.Block, error) {
	block, err := getBlock(api.s.chain, height)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// TpsInfo tps detail info
type TpsInfo struct {
	StartHeight uint64
	EndHeight   uint64
	Count       uint64
	Duration    uint64
}

// GetTPS get tps info
func (api *PrivateDebugAPI) GetTPS() (*TpsInfo, error) {
	chain := api.s.BlockChain()
	block := chain.CurrentBlock()
	timeInterval := uint64(150)
	if block.Header.Height == 0 {
		return nil, nil
	}

	startHeight := int64(block.Header.Height) - 6
	// genesis block time is set by users, so must calculate from the number that greater than 1
	if startHeight < 2 {
		return nil, nil
	}

	count := uint64(0)
	duration := uint64(0)
	var endHeight uint64

	for height := uint64(startHeight); height > 1; height-- {
		current, err := chain.GetStore().GetBlockByHeight(height)
		if err != nil {
			return nil, fmt.Errorf("failed to get block, error:%s, block height:%d", err, height)
		}

		for _, tx := range current.Transactions {
			if !tx.IsCrossShardTx() {
				count = count + 1
			}
		}

		count = count + uint64(len(current.Debts)) - 1
		front, err := chain.GetStore().GetBlockByHeight(height - 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get block, error:%s, block height:%d", err, height-1)
		}

		duration += current.Header.CreateTimestamp.Uint64() - front.Header.CreateTimestamp.Uint64()
		endHeight = height

		if duration > timeInterval {
			break
		}
	}

	return &TpsInfo{
		StartHeight: endHeight,
		EndHeight:   uint64(startHeight),
		Count:       count,
		Duration:    duration,
	}, nil
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

// GetPendingDebts returns all pending debts
func (api *PrivateDebugAPI) GetPendingDebts() ([]*types.Debt, error) {
	return api.s.DebtPool().GetDebts(false, true), nil
}
