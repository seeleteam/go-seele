/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
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

// MinerInfo miner simple info
type MinerInfo struct {
	Coinbase           common.Address
	CurrentBlockHeight uint64
	HeaderHash         common.Hash
	Shard              uint
	MinerStatus        string
	MinerThread        int
}

// GetBalanceResponse response param for GetBalance api
type GetBalanceResponse struct {
	Account common.Address
	Balance *big.Int
}

// GetBlockByHeightRequest request param for GetBlockByHeight api
type GetBlockByHeightRequest struct {
	Height int64
	FullTx bool
}

// GetBlocksRequest request param for GetBlocks api
type GetBlocksRequest struct {
	GetBlockByHeightRequest
	Size uint
}

// GetBlockByHashRequest request param for GetBlockByHash api
type GetBlockByHashRequest struct {
	HashHex string
	FullTx  bool
}

// GetLogsRequest request param for GetLogs api
type GetLogsRequest struct {
	Height          int64
	ContractAddress string
	Topics          string
}

// GetLogsResponse response param for GetLogs api
type GetLogsResponse struct {
	Txhash   common.Hash
	LogIndex uint
	Log      *types.Log
}

// GetTxByBlockHeightAndIndexRequest request param for GetTransactionByBlockHeightAndIndex api
type GetTxByBlockHeightAndIndexRequest struct {
	Height int64
	Index  uint
}

// GetTxByBlockHashAndIndexRequest request param for GetTransactionByBlockHashAndIndex api
type GetTxByBlockHashAndIndexRequest struct {
	HashHex string
	Index   uint
}

// CallRequest request param for Call api
type CallRequest struct {
	Tx     *types.Transaction
	Height int64
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

// GetInfo gets the account address that mining rewards will be send to.
func (api *PublicSeeleAPI) GetInfo() (MinerInfo, error) {
	block := api.s.chain.CurrentBlock()

	var status string
	if api.s.miner.IsMining() {
		status = "Running"
	} else {
		status = "Stopped"
	}

	return MinerInfo{
		Coinbase:           api.s.miner.GetCoinbase(),
		CurrentBlockHeight: block.Header.Height,
		HeaderHash:         block.HeaderHash,
		Shard:              common.LocalShardNumber,
		MinerStatus:        status,
		MinerThread:        api.s.miner.GetThreads(),
	}, nil
}

// GetBalance get balance of the account. if the account's address is empty, will get the coinbase balance
func (api *PublicSeeleAPI) GetBalance(account common.Address) (*GetBalanceResponse, error) {
	if account.Equal(common.EmptyAddress) {
		account = api.s.Miner().GetCoinbase()
	}

	state, err := api.s.chain.GetCurrentState()
	if err != nil {
		return nil, err
	}

	return &GetBalanceResponse{
		Account: account,
		Balance: state.GetBalance(account),
	}, nil
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

// GetAccountNonce get account next used nonce
func (api *PublicSeeleAPI) GetAccountNonce(account common.Address) (uint64, error) {
	if account.Equal(common.EmptyAddress) {
		account = api.s.Miner().GetCoinbase()
	}

	state, err := api.s.chain.GetCurrentState()
	if err != nil {
		return 0, err
	}

	return state.GetNonce(account), nil
}

// GetBlockHeight get the block height of the chain head
func (api *PublicSeeleAPI) GetBlockHeight() (uint64, error) {
	block := api.s.chain.CurrentBlock()
	return block.Header.Height, nil
}

// GetBlockByHeight returns the requested block. When blockNr is -1 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlockByHeight(height int64, fulltx bool) (map[string]interface{}, error) {
	block, err := getBlock(api.s.chain, height)
	if err != nil {
		return nil, err
	}

	return rpcOutputBlock(block, fulltx, api.s.chain.GetStore())
}

// GetBlocks returns the size of requested block. When the blockNr is -1 the chain head is returned.
//When the size is greater than 64, the size will be set to 64.When it's -1 that the blockNr minus size, the blocks in 64 is returned.
// When fullTx is true all transactions in the block are returned in full detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlocks(height int64, fulltx bool, size uint) ([]map[string]interface{}, error) {
	blocks := make([]types.Block, 0)
	if height < 0 {
		block := api.s.chain.CurrentBlock()
		blocks = append(blocks, *block)
	} else {
		if size > maxSizeLimit {
			size = maxSizeLimit
		}

		if height+1-int64(size) < 0 {
			size = uint(height + 1)
		}

		for i := uint(0); i < size; i++ {
			var block *types.Block
			block, err := getBlock(api.s.chain, height-int64(i))
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, *block)
		}
	}

	return rpcOutputBlocks(blocks, fulltx, api.s.chain.GetStore())
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlockByHash(hashHex string, fulltx bool) (map[string]interface{}, error) {
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

	return rpcOutputBlock(block, fulltx, store)
}

// GetLogs Get the logs that satisfies the condition in the block by height and filter
func (api *PublicSeeleAPI) GetLogs(height int64, contract string, topics string) ([]GetLogsResponse, error) {
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

	logs := make([]GetLogsResponse, 0)
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

			logs = append(logs, GetLogsResponse{
				Txhash:   receipt.TxHash,
				LogIndex: uint(logIndex),
				Log:      log,
			})
		}
	}

	return logs, nil
}

// rpcOutputBlock converts the given block to the RPC output which depends on fullTx
func rpcOutputBlock(b *types.Block, fullTx bool, store store.BlockchainStore) (map[string]interface{}, error) {
	head := b.Header
	fields := map[string]interface{}{
		"height":     head.Height,
		"hash":       b.HeaderHash.ToHex(),
		"parentHash": head.PreviousBlockHash.ToHex(),
		"nonce":      head.Nonce,
		"stateHash":  head.StateHash.ToHex(),
		"txHash":     head.TxHash.ToHex(),
		"creator":    head.Creator.ToHex(),
		"timestamp":  head.CreateTimestamp,
		"difficulty": head.Difficulty,
	}

	txs := b.Transactions
	transactions := make([]interface{}, len(txs))
	for i, tx := range txs {
		if fullTx {
			transactions[i] = PrintableOutputTx(tx)
		} else {
			transactions[i] = tx.Hash.ToHex()
		}
	}
	fields["transactions"] = transactions

	totalDifficulty, err := store.GetBlockTotalDifficulty(b.HeaderHash)
	if err != nil {
		return nil, err
	}
	fields["totalDifficulty"] = totalDifficulty

	return fields, nil
}

func rpcOutputBlocks(b []types.Block, fullTx bool, store store.BlockchainStore) ([]map[string]interface{}, error) {
	fields := make([]map[string]interface{}, 0)

	for i := range b {
		if field, err := rpcOutputBlock(&b[i], fullTx, store); err == nil {
			fields = append(fields, field)
		}
	}
	return fields, nil
}

// PrintableOutputTx converts the given tx to the RPC output
func PrintableOutputTx(tx *types.Transaction) map[string]interface{} {
	toAddr := ""
	if !tx.Data.To.IsEmpty() {
		toAddr = tx.Data.To.ToHex()
	}

	transaction := map[string]interface{}{
		"hash":         tx.Hash.ToHex(),
		"from":         tx.Data.From.ToHex(),
		"to":           toAddr,
		"amount":       tx.Data.Amount,
		"accountNonce": tx.Data.AccountNonce,
		"payload":      tx.Data.Payload,
		"timestamp":    tx.Data.Timestamp,
		"fee":          tx.Data.Fee,
	}
	return transaction
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

// getBlock returns block by height,when height is -1 the chain head is returned
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
