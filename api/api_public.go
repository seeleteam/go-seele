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
	block := api.s.Chain().CurrentBlock()

	var status string
	if api.s.IsMining() {
		status = "Running"
	} else {
		status = "Stopped"
	}

	return GetMinerInfo{
		Coinbase:           api.s.GetCoinbase(),
		CurrentBlockHeight: block.Header.Height,
		HeaderHash:         block.HeaderHash,
		Shard:              common.LocalShardNumber,
		MinerStatus:        status,
		MinerThread:        api.s.GetThreads(),
	}, nil
}

func (api *PublicSeeleAPI) GetBalance(account common.Address) (*GetBalanceResponse, error) {
	if account.Equal(common.EmptyAddress) {
		account = api.s.GetCoinbase()
	}

	balance, err := api.s.Chain().GetCurrentStateBalance(account)
	if err != nil {
		return nil, err
	}

	return &GetBalanceResponse{
		Account: account,
		Balance: balance,
	}, nil
}

// GetAccountNonce get account next used nonce
func (api *PublicSeeleAPI) GetAccountNonce(account common.Address) (uint64, error) {
	if account.Equal(common.EmptyAddress) {
		account = api.s.GetCoinbase()
	}

	nonce, err := api.s.Chain().GetCurrentStateNonce()
	if err != nil {
		return 0, err
	}

	return nonce, nil
}

// GetBlockHeight get the block height of the chain head
func (api *PublicSeeleAPI) GetBlockHeight() (uint64, error) {
	block := api.s.Chain().CurrentBlock()
	return block.Header.Height, nil
}

// GetBlock returns the requested block.
func (api *PublicSeeleAPI) GetBlock(hashHex string, height int64, fulltx bool) (map[string]interface{}, error) {
	if len(hashHex) > 0 {
		return api.GetBlockByHash(hashHex, fulltx)
	}

	return api.GetBlockByHeight(height, fulltx)
}

// GetBlockByHeight returns the requested block. When blockNr is -1 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlockByHeight(height int64, fulltx bool) (map[string]interface{}, error) {
	block, err := getBlock(api.s.Chain(), height)
	if err != nil {
		return nil, err
	}

	return rpcOutputBlock(block, fulltx, api.s.Chain().GetStore())
}

// getBlock returns block by height,when height is -1 the chain head is returned
func getBlock(chain Chain, height int64) (*types.Block, error) {
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

// GetBlocks returns the size of requested block. When the blockNr is -1 the chain head is returned.
//When the size is greater than 64, the size will be set to 64.When it's -1 that the blockNr minus size, the blocks in 64 is returned.
// When fullTx is true all transactions in the block are returned in full detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlocks(height int64, fulltx bool, size uint) ([]map[string]interface{}, error) {
	blocks := make([]types.Block, 0)
	if height < 0 {
		block := api.s.Chain().CurrentBlock()
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
			block, err := getBlock(api.s.Chain(), height-int64(i))
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, *block)
		}
	}

	return rpcOutputBlocks(blocks, fulltx, api.s.Chain().GetStore())
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlockByHash(hashHex string, fulltx bool) (map[string]interface{}, error) {
	store := api.s.Chain().GetStore()
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

// @todo AddTx add a tx to miner
func (api *PublicSeeleAPI) AddTx(tx types.Transaction) (bool, error) {
	return false, nil
}
