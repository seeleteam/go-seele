/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"math/big"

	"github.com/davecgh/go-spew/spew"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
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
func (api *PrivateDebugAPI) PrintBlock(height *int64, result *string) error {
	block, err := getBlock(api.s.chain, *height)
	if err != nil {
		return err
	}

	*result = spew.Sdump(block)
	return nil
}

// GetTxPoolContent returns the transactions contained within the transaction pool
func (api *PrivateDebugAPI) GetTxPoolContent(input interface{}, result *map[string][]map[string]interface{}) error {
	txPool := api.s.TxPool()
	data := txPool.GetProcessableTransactions()

	content := make(map[string][]map[string]interface{})
	for adress, txs := range data {
		trans := make([]map[string]interface{}, len(txs))
		for i, tran := range txs {
			trans[i] = PrintableOutputTx(tran)
		}
		content[adress.ToHex()] = trans
	}
	*result = content

	return nil
}

// GetTxPoolTxCount returns the number of transaction in the pool
func (api *PrivateDebugAPI) GetTxPoolTxCount(input interface{}, result *uint64) error {
	txPool := api.s.TxPool()
	*result = uint64(txPool.GetProcessableTransactionsCount())
	return nil
}

func (api *PrivateDebugAPI) GetTPS(input interface{}, result *uint64) error {
	chain := api.s.BlockChain()
	block, _ := chain.CurrentBlock()
	if block.Header.Height < 2 {
		*result = 0
		return nil
	}

	firstBlock, err := chain.GetStore().GetBlockByHeight(1)
	if err != nil {
		return err
	}

	durationBig := big.NewInt(0)
	durationBig.Sub(block.Header.CreateTimestamp, firstBlock.Header.CreateTimestamp)
	mins10 := uint64(10 * 60)

	var count = uint64(len(block.Transactions) - 1)
	var duration uint64
	if durationBig.Uint64() > mins10 {
		duration = durationBig.Uint64()
		for height := block.Header.Height - 1; height > 0; height-- {
			current, err := chain.GetStore().GetBlockByHeight(height)
			if err != nil {
				return err
			}

			count += uint64(len(current.Transactions) - 1)
		}
	} else {
		startBig := block.Header.CreateTimestamp.Uint64()
		for height := block.Header.Height - 1; height > 0; height-- {
			current, err := chain.GetStore().GetBlockByHeight(height)
			if err != nil {
				return err
			}

			count += uint64(len(current.Transactions) - 1)
			duration = startBig - current.Header.CreateTimestamp.Uint64()
			if duration > mins10 {
				break
			}
		}
	}

	*result = count
	return nil
}
