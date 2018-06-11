/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
)

// PublicSeeleAPI provides an API to access full node-related information.
type PublicSeeleAPI struct {
	s *SeeleService
}

// NewPublicSeeleAPI creates a new PublicSeeleAPI object for rpc service.
func NewPublicSeeleAPI(s *SeeleService) *PublicSeeleAPI {
	return &PublicSeeleAPI{s}
}

// MinerInfo miner simple info
type MinerInfo struct {
	Coinbase           common.Address
	CurrentBlockHeight uint64
	HeaderHash         common.Hash
}

// GetBlockByHeightRequest request param for GetBlockByHeight api
type GetBlockByHeightRequest struct {
	Height int64
	FullTx bool
}

// GetBlockByHashRequest request param for GetBlockByHash api
type GetBlockByHashRequest struct {
	HashHex string
	FullTx  bool
}

// GetTxByBlockHeightAndIndexRequest request param for GetTransactionByBlockHeightAndIndex api
type GetTxByBlockHeightAndIndexRequest struct {
	Height int64
	Index  int
}

// GetTxByBlockHashAndIndexRequest request param for GetTransactionByBlockHashAndIndex api
type GetTxByBlockHashAndIndexRequest struct {
	HashHex string
	Index   int
}

// GetInfo gets the account address that mining rewards will be send to.
func (api *PublicSeeleAPI) GetInfo(input interface{}, info *MinerInfo) error {
	block, _ := api.s.chain.CurrentBlock()

	*info = MinerInfo{
		Coinbase:           api.s.miner.GetCoinbase(),
		CurrentBlockHeight: block.Header.Height,
		HeaderHash:         block.HeaderHash,
	}

	return nil
}

// GetBalance get balance of the account. if the account's address is empty, will get the coinbase balance
func (api *PublicSeeleAPI) GetBalance(account *common.Address, result *big.Int) error {
	if account == nil || account.Equal(common.Address{}) {
		*account = api.s.Miner().GetCoinbase()
	}

	state := api.s.chain.CurrentState()
	balance := state.GetBalance(*account)
	result.Set(balance)
	return nil
}

// AddTx add a tx to miner
func (api *PublicSeeleAPI) AddTx(tx *types.Transaction, result *bool) error {
	shard := tx.Data.From.Shard()
	var err error
	if shard != common.LocalShardNumber {
		if err = tx.ValidateWithoutState(); err == nil {
			api.s.seeleProtocol.SendDifferentShardTx(tx, shard)
		}
	} else {
		err = api.s.txPool.AddTransaction(tx)
	}

	if err != nil {
		*result = false
		return err
	}

	*result = true
	return nil
}

// GetAccountNonce get account next used nonce
func (api *PublicSeeleAPI) GetAccountNonce(account *common.Address, nonce *uint64) error {
	state := api.s.chain.CurrentState()
	*nonce = state.GetNonce(*account)

	return nil
}

// GetBlockHeight get the block height of the chain head
func (api *PublicSeeleAPI) GetBlockHeight(input interface{}, height *uint64) error {
	block, _ := api.s.chain.CurrentBlock()
	*height = block.Header.Height

	return nil
}

// GetBlockByHeight returns the requested block. When blockNr is -1 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlockByHeight(request *GetBlockByHeightRequest, result *map[string]interface{}) error {
	block, err := getBlock(api.s.chain, request.Height)
	if err != nil {
		return err
	}

	response, err := rpcOutputBlock(block, request.FullTx, api.s.chain.GetStore())
	if err != nil {
		return err
	}

	*result = response
	return nil
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlockByHash(request *GetBlockByHashRequest, result *map[string]interface{}) error {
	store := api.s.chain.GetStore()
	hashByte, err := hexutil.HexToBytes(request.HashHex)
	if err != nil {
		return err
	}

	hash := common.BytesToHash(hashByte)
	block, err := store.GetBlock(hash)
	if err != nil {
		return err
	}

	response, err := rpcOutputBlock(block, request.FullTx, store)
	if err != nil {
		return err
	}

	*result = response
	return nil
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

// PrintableOutputTx converts the given tx to the RPC output
func PrintableOutputTx(tx *types.Transaction) map[string]interface{} {
	transaction := map[string]interface{}{
		"hash":         tx.Hash.ToHex(),
		"from":         tx.Data.From.ToHex(),
		"to":           tx.Data.To.ToHex(),
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
	contractAddr, err := common.NewAddress(re.ContractAddress)
	if err != nil {
		return nil, err
	}

	outMap := map[string]interface{}{
		"result":    hexutil.BytesToHex(re.Result),
		"poststate": re.PostState.ToHex(),
		"txhash":    re.TxHash.ToHex(),
		"contract":  contractAddr.ToHex(),
	}
	return outMap, nil
}

// getBlock returns block by height,when height is -1 the chain head is returned
func getBlock(chain *core.Blockchain, height int64) (*types.Block, error) {
	var block *types.Block
	if height == -1 {
		block, _ = chain.CurrentBlock()
	} else {
		var err error
		block, err = chain.GetStore().GetBlockByHeight(uint64(height))
		if err != nil {
			return nil, err
		}
	}

	return block, nil
}
