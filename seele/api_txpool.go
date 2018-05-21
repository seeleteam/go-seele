/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"errors"
	"strconv"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
)

// PublicTransactionPoolAPI provides an API to access transaction pool information.
type PublicTransactionPoolAPI struct {
	s *SeeleService
}

// NewPublicTransactionPoolAPI creates a new PublicTransactionPoolAPI object for transaction pool rpc service.
func NewPublicTransactionPoolAPI(s *SeeleService) *PublicTransactionPoolAPI {
	return &PublicTransactionPoolAPI{s}
}

// GetBlockTransactionCountByHeight returns the count of transactions in the block with the given height.
func (api *PublicTransactionPoolAPI) GetBlockTransactionCountByHeight(height *int64, result *int) error {
	block, err := getBlock(api.s.chain, *height)
	if err != nil {
		return err
	}
	*result = len(block.Transactions)
	return nil
}

// GetBlockTransactionCountByHash returns the count of transactions in the block with the given hash.
func (api *PublicTransactionPoolAPI) GetBlockTransactionCountByHash(blockHash *string, result *int) error {
	store := api.s.chain.GetStore()
	hashByte, err := hexutil.HexToBytes(*blockHash)
	if err != nil {
		return err
	}

	hash := common.BytesToHash(hashByte)
	block, err := store.GetBlock(hash)
	if err != nil {
		return err
	}
	*result = len(block.Transactions)
	return nil
}

// GetTransactionByBlockHeightAndIndex returns the transaction in the block with the given block height and index.
func (api *PublicTransactionPoolAPI) GetTransactionByBlockHeightAndIndex(request *GetTxByBlockHeightAndIndexRequest, result *map[string]interface{}) error {
	block, err := getBlock(api.s.chain, request.Height)
	if err != nil {
		return err
	}

	txs := block.Transactions
	if request.Index >= len(txs) {
		return errors.New(errIndexOutRange.Error() + strconv.Itoa(len(txs)-1))
	}

	*result = rpcOutputTx(txs[request.Index])
	return nil
}

// GetTransactionByBlockHashAndIndex returns the transaction in the block with the given block hash and index.
func (api *PublicTransactionPoolAPI) GetTransactionByBlockHashAndIndex(request *GetTxByBlockHashAndIndexRequest, result *map[string]interface{}) error {
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

	txs := block.Transactions
	if request.Index >= len(txs) {
		return errors.New(errIndexOutRange.Error() + strconv.Itoa(len(txs)-1))
	}
	*result = rpcOutputTx(txs[request.Index])
	return nil
}
