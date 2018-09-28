/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package api

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
)

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
		"poststate": re.PostState.ToHex(),
		"txhash":    re.TxHash.ToHex(),
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

		outMap["contract"] = contractAddr.ToHex()
	}

	if len(re.Logs) > 0 {
		var logOuts []map[string]interface{}

		for _, log := range re.Logs {
			logOut, err := printableLog(log)
			if err != nil {
				return nil, err
			}

			logOuts = append(logOuts, logOut)
		}

		outMap["logs"] = logOuts
	}

	return outMap, nil
}

func printableLog(log *types.Log) (map[string]interface{}, error) {
	if (len(log.Data) % 32) > 0 {
		return nil, fmt.Errorf("invalid log data length %v", len(log.Data))
	}

	outMap := map[string]interface{}{
		"address": log.Address.ToHex(),
	}

	// data
	dataLen := len(log.Data) / 32
	if dataLen > 0 {
		var data []string
		for i := 0; i < dataLen; i++ {
			data = append(data, hexutil.BytesToHex(log.Data[i*32:(i+1)*32]))
		}
		outMap["data"] = data
	}

	// topics
	switch len(log.Topics) {
	case 0:
		// do not print empty topic
	case 1:
		outMap["topic"] = log.Topics[0].ToHex()
	default:
		var topics []string
		for _, t := range log.Topics {
			topics = append(topics, t.ToHex())
		}
		outMap["topics"] = fmt.Sprintf("[%v]", strings.Join(topics, ", "))
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

// GetTransactionByBlockIndex returns the transaction in the block with the given block hash/height and index.
func (api *TransactionPoolAPI) GetTransactionByBlockIndex(hashHex string, height int64, index uint) (map[string]interface{}, error) {
	if len(hashHex) > 0 {
		return api.GetTransactionByBlockHashAndIndex(hashHex, index)
	}

	return api.GetTransactionByBlockHeightAndIndex(height, index)
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
func (api *TransactionPoolAPI) GetReceiptByTxHash(txHash string) (map[string]interface{}, error) {
	hash, err := common.HexToHash(txHash)
	if err != nil {
		return nil, err
	}

	receipt, err := api.s.GetReceiptByTxHash(hash)
	if err != nil {
		return nil, err
	}

	return PrintableReceipt(receipt)
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

	debt := types.NewDebt(tx)
	if debt != nil {
		output["debt"] = debt
	}

	if idx == nil {
		output["status"] = "pool"
	} else {
		output["status"] = "block"

		output["blockHash"] = idx.BlockHash.ToHex()
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
