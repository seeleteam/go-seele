/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"fmt"
	"math/big"
	"strings"

	api2"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
)

// PublicSeeleAPI provides an API to access full node-related information.
type PublicSeeleAPI struct {
	s *SeeleService
}

const maxSizeLimit = 64

// NewPublicSeeleAPI creates a new PublicSeeleAPI object for rpc service.
func NewPublicSeeleAPI(s *SeeleService) *PublicSeeleAPI {
	return &PublicSeeleAPI{s}
}

// Call is to execute a given transaction on a statedb of a given block height.
// It does not affect this statedb and blockchain and is useful for executing and retrieve values.
func (api *PublicSeeleAPI) Call(contract, payload string, height int64) (map[string]interface{}, error) {
	contractAddr, err := common.HexToAddress(contract)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %s", err)
	}

	msg, err := hexutil.HexToBytes(payload)
	if err != nil {
		return nil, fmt.Errorf("invalid payload, %s", err)
	}

	// Get the block by block height, if the height is less than zero, get the current block.
	block, err := getBlock(api.s.chain, height)
	if err != nil {
		return nil, err
	}

	// Get the statedb by the given block height
	statedb, err := state.NewStatedb(block.Header.StateHash, api.s.accountStateDB)
	if err != nil {
		return nil, err
	}

	coinbase := api.s.miner.GetCoinbase()
	from := crypto.MustGenerateShardAddress(coinbase.Shard())
	statedb.CreateAccount(*from)
	statedb.SetBalance(*from, common.SeeleToFan)

	amount, fee, nonce := big.NewInt(0), big.NewInt(1), uint64(1)
	tx, err := types.NewMessageTransaction(*from, contractAddr, amount, fee, nonce, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %s", err)
	}

	// Get the transaction receipt, and the fee give to the miner coinbase
	receipt, err := api.s.chain.ApplyTransaction(tx, 0, coinbase, statedb, block.Header)
	if err != nil {
		return nil, err
	}

	// Format the receipt
	result, err := PrintableReceipt(receipt)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// AddTx add a tx to miner
func (api *PublicSeeleAPI) AddTx(tx types.Transaction) (bool, error) {
	shard := tx.Data.From.Shard()
	var err error
	if shard != common.LocalShardNumber {
		if err = tx.ValidateWithoutState(true, false); err == nil {
			api.s.seeleProtocol.SendDifferentShardTx(&tx, shard)
		}
	} else {
		err = api.s.txPool.AddTransaction(&tx)
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

// GetLogs Get the logs that satisfies the condition in the block by height and filter
func (api *PublicSeeleAPI) GetLogs(height int64, contract string, topics string) ([]api2.GetLogsResponse, error) {
	// Check input parameters
	contractAddress, err := common.HexToAddress(contract)
	if err != nil {
		return nil, fmt.Errorf("Invalid contract address, %s", err)
	}

	hash, err := common.HexToHash(topics)
	if err != nil {
		return nil, fmt.Errorf("Invalid topic, %s", err)
	}

	// Do filter
	block, err := getBlock(api.s.chain, height)
	if err != nil {
		return nil, err
	}

	store := api.s.chain.GetStore()
	receipts, err := store.GetReceiptsByBlockHash(block.HeaderHash)
	if err != nil {
		return nil, err
	}

	logs := make([]api2.GetLogsResponse, 0)
	for _, receipt := range receipts {
		for logIndex, log := range receipt.Logs {
			// Matches contract address
			if !contractAddress.Equal(log.Address) {
				continue
			}

			// Matches topics
			// Because of the topics is always only one
			if len(log.Topics) < 1 || !hash.Equal(log.Topics[0]) {
				continue
			}

			logs = append(logs, api2.GetLogsResponse{
				Txhash:   receipt.TxHash,
				LogIndex: uint(logIndex),
				Log:      log,
			})
		}
	}

	return logs, nil
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

// getBlock returns block by height,when height is less than 0 the chain head is returned
func getBlock(chain *core.Blockchain, height int64) (*types.Block, error) {
	var block *types.Block
	if height < 0 {
		block = chain.CurrentBlock()
	} else {
		var err error
		block, err = chain.GetStore().GetBlockByHeight(uint64(height))
		if err != nil {
			return nil, err
		}
	}

	return block, nil
}
