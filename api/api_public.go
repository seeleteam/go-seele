/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package api

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
)

const maxSizeLimit = 64

// PublicSeeleAPI provides an API to access full node-related information.
type PublicSeeleAPI struct {
	s Backend
}

// NewPublicSeeleAPI creates a new PublicSeeleAPI object for rpc service.
func NewPublicSeeleAPI(s Backend) *PublicSeeleAPI {
	return &PublicSeeleAPI{s}
}

// GetInfo gets the account address that mining rewards will be send to.
func (api *PublicSeeleAPI) GetInfo() (GetMinerInfo, error) {
	header := api.s.ChainBackend().CurrentHeader()
	var status string
	if api.s.IsMining() {
		status = "Running"
	} else {
		status = "Stopped"
	}

	return GetMinerInfo{
		Coinbase:           api.s.GetMinerCoinbase(),
		CurrentBlockHeight: header.Height,
		HeaderHash:         header.Hash(),
		Shard:              common.LocalShardNumber,
		MinerStatus:        status,
		MinerThread:        api.s.GetThreads(),
	}, nil
}

// GetBalance get balance of the account. if the account's address is empty, will get the coinbase balance
func (api *PublicSeeleAPI) GetBalance(account common.Address) (*GetBalanceResponse, error) {
	if account.IsEmpty() {
		account = api.s.GetMinerCoinbase()
	}

	state, err := api.s.ChainBackend().GetCurrentState()
	if err != nil {
		return nil, err
	}

	return &GetBalanceResponse{
		Account: account,
		Balance: state.GetBalance(account),
	}, nil
}

// GetAccountNonce get account next used nonce
func (api *PublicSeeleAPI) GetAccountNonce(account common.Address) (uint64, error) {
	if account.Equal(common.EmptyAddress) {
		account = api.s.GetMinerCoinbase()
	}

	state, err := api.s.ChainBackend().GetCurrentState()
	if err != nil {
		return 0, err
	}

	return state.GetNonce(account), nil
}

// GetBlockHeight get the block height of the chain head
func (api *PublicSeeleAPI) GetBlockHeight() (uint64, error) {
	header := api.s.ChainBackend().CurrentHeader()
	return header.Height, nil
}

// GetBlock returns the requested block.
func (api *PublicSeeleAPI) GetBlock(hashHex string, height int64, fulltx bool) (map[string]interface{}, error) {
	if len(hashHex) > 0 {
		return api.GetBlockByHash(hashHex, fulltx)
	}

	return api.GetBlockByHeight(height, fulltx)
}

// GetBlockByHeight returns the requested block. When blockNr is less than 0 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlockByHeight(height int64, fulltx bool) (map[string]interface{}, error) {
	block, err := getBlock(api.s.ChainBackend(), height)
	if err != nil {
		return nil, err
	}

	return rpcOutputBlock(block, fulltx, api.s.ChainBackend().GetStore())
}

// getBlock returns block by height,when height is less than 0 the chain head is returned
func getBlock(chain Chain, height int64) (block *types.Block, err error) {
	if height < 0 {
		header := chain.CurrentHeader()
		block, err = chain.GetStore().GetBlockByHeight(header.Height)
	} else {
		var err error
		block, err = chain.GetStore().GetBlockByHeight(uint64(height))
		if err != nil {
			return nil, err
		}
	}

	return block, nil
}

// GetBlocks returns the size of requested block. When the blockNr is -1 the chain head is returned.
//When the size is greater than 64, the size will be set to 64.When it's -1 that the blockNr minus size, the blocks in 64 is returned.
// When fullTx is true all transactions in the block are returned in full detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlocks(height int64, fulltx bool, size uint) ([]map[string]interface{}, error) {
	blocks := make([]types.Block, 0)
	if height < 0 {
		header := api.s.ChainBackend().CurrentHeader()
		block, err := api.s.ChainBackend().GetStore().GetBlockByHeight(header.Height)
		if err != nil {
			return nil, err
		}

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
			block, err := getBlock(api.s.ChainBackend(), height-int64(i))
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, *block)
		}
	}

	return rpcOutputBlocks(blocks, fulltx, api.s.ChainBackend().GetStore())
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlockByHash(hashHex string, fulltx bool) (map[string]interface{}, error) {
	store := api.s.ChainBackend().GetStore()
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

// rpcOutputBlock converts the given block to the RPC output which depends on fullTx
func rpcOutputBlock(b *types.Block, fullTx bool, store store.BlockchainStore) (map[string]interface{}, error) {
	head := b.Header
	fields := map[string]interface{}{
		"header": head,
		"hash":   b.HeaderHash.ToHex(),
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

	debts := types.NewDebts(txs)
	fields["txDebts"] = getOutputDebts(debts, fullTx)
	fields["debts"] = getOutputDebts(b.Debts, fullTx)

	return fields, nil
}

func getOutputDebts(debts []*types.Debt, fullTx bool) []interface{} {
	outputDebts := make([]interface{}, len(debts))
	for i, d := range debts {
		if fullTx {
			outputDebts[i] = d
		} else {
			outputDebts[i] = d.Hash
		}
	}

	return outputDebts
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
