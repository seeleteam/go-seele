/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package api

import (
	"fmt"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

// ErrInvalidAccount the account is invalid
var ErrInvalidAccount = errors.New("invalid account")

const (
	maxSizeLimit = 64
)

// PublicSeeleAPI provides an API to access full node-related information.
type PublicSeeleAPI struct {
	s Backend
}

// NewPublicSeeleAPI creates a new PublicSeeleAPI object for rpc service.
func NewPublicSeeleAPI(s Backend) *PublicSeeleAPI {
	return &PublicSeeleAPI{s}
}

// GetBalance get balance of the account. if the account's address is empty, will get the coinbase balance
func (api *PublicSeeleAPI) GetBalance(account common.Address) (*GetBalanceResponse, error) {
	if account.IsEmpty() {
		return nil, ErrInvalidAccount
	}

	state, err := api.s.ChainBackend().GetCurrentState()
	if err != nil {
		return nil, err
	}

	var info GetBalanceResponse
	// is local shard?
	if common.LocalShardNumber != account.Shard() {
		return nil, fmt.Errorf("local shard is: %d, your shard is: %d, you need to change to shard %d to get your balance", common.LocalShardNumber, account.Shard(), account.Shard())
	}

	info.Balance = state.GetBalance(account)
	info.Account = account

	return &info, nil
}

// GetAccountNonce get account next used nonce
func (api *PublicSeeleAPI) GetAccountNonce(account common.Address) (uint64, error) {
	if account.Equal(common.EmptyAddress) {
		return 0, ErrInvalidAccount
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
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return nil, err
	}
	totalDifficulty, err := api.s.GetBlockTotalDifficulty(block.HeaderHash)
	if err != nil {
		return nil, err
	}
	return rpcOutputBlock(block, fulltx, totalDifficulty)
}

// GetBlocks returns the size of requested block. When the blockNr is -1 the chain head is returned.
//When the size is greater than 64, the size will be set to 64.When it's -1 that the blockNr minus size, the blocks in 64 is returned.
// When fullTx is true all transactions in the block are returned in full detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlocks(height int64, fulltx bool, size uint) ([]map[string]interface{}, error) {
	blocks := make([]*types.Block, 0)
	totalDifficultys := make([]*big.Int, 0)
	if height < 0 {
		header := api.s.ChainBackend().CurrentHeader()
		block, err := api.s.GetBlock(common.EmptyHash, int64(header.Height))
		if err != nil {
			return nil, err
		}
		totalDifficulty, err := api.s.GetBlockTotalDifficulty(block.HeaderHash)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
		totalDifficultys = append(totalDifficultys, totalDifficulty)
	} else {
		if size > maxSizeLimit {
			size = maxSizeLimit
		}

		if height+1-int64(size) < 0 {
			size = uint(height + 1)
		}

		for i := uint(0); i < size; i++ {
			var block *types.Block
			block, err := api.s.GetBlock(common.EmptyHash, height-int64(i))
			if err != nil {
				return nil, err
			}
			totalDifficulty, err := api.s.GetBlockTotalDifficulty(block.HeaderHash)
			if err != nil {
				return nil, err
			}
			totalDifficultys = append(totalDifficultys, totalDifficulty)
			blocks = append(blocks, block)
		}
	}

	return rpcOutputBlocks(blocks, fulltx, totalDifficultys)
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlockByHash(hashHex string, fulltx bool) (map[string]interface{}, error) {
	hash, err := common.HexToHash(hashHex)
	if err != nil {
		return nil, err
	}

	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return nil, err
	}

	totalDifficulty, err := api.s.GetBlockTotalDifficulty(block.HeaderHash)
	if err != nil {
		return nil, err
	}
	return rpcOutputBlock(block, fulltx, totalDifficulty)
}

// rpcOutputBlock converts the given block to the RPC output which depends on fullTx
func rpcOutputBlock(b *types.Block, fullTx bool, totalDifficulty *big.Int) (map[string]interface{}, error) {
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

func rpcOutputBlocks(b []*types.Block, fullTx bool, d []*big.Int) ([]map[string]interface{}, error) {
	fields := make([]map[string]interface{}, 0)

	for i := range b {
		if field, err := rpcOutputBlock(b[i], fullTx, d[i]); err == nil {
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
		"gasPrice":     tx.Data.GasPrice,
		"gasLimit":     tx.Data.GasLimit,
	}
	return transaction
}

// AddTx add a tx to miner
func (api *PublicSeeleAPI) AddTx(tx types.Transaction) (bool, error) {
	shard := tx.Data.From.Shard()
	var err error
	if shard != common.LocalShardNumber {
		if err = tx.ValidateWithoutState(true, false); err == nil {
			api.s.ProtocolBackend().SendDifferentShardTx(&tx, shard)
		}
	} else {
		err = api.s.TxPoolBackend().AddTransaction(&tx)
	}

	if err != nil {
		return false, err
	}
	api.s.Log().Debug("create transaction and add it. transaction hash: %v, time: %d", tx.Hash, time.Now().UnixNano())
	return true, nil
}
