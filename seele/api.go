/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"math/big"
	"strconv"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
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

// GetInfo gets the account address that mining rewards will be send to.
func (api *PublicSeeleAPI) GetInfo(input interface{}, info *MinerInfo) error {
	block, _ := api.s.chain.CurrentBlock()

	*info = MinerInfo{
		Coinbase:           api.s.Coinbase,
		CurrentBlockHeight: block.Header.Height,
		HeaderHash:         block.HeaderHash,
	}

	return nil
}

// GetBalance get balance of the account. if the account's address is empty, will get the coinbase balance
func (api *PublicSeeleAPI) GetBalance(account *common.Address, result *big.Int) error {
	if account == nil || account.Equal(common.Address{}) {
		*account = api.s.Coinbase
	}

	state := api.s.chain.CurrentState()
	amount, _ := state.GetAmount(*account)
	result.Set(amount)
	return nil
}

// AddTx add a tx to miner
func (api *PublicSeeleAPI) AddTx(tx *types.Transaction, result *bool) error {
	err := api.s.txPool.AddTransaction(tx)
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
	*nonce, _ = state.GetNonce(*account)

	return nil
}

// GetBlockNumber get the block number of the chain head
func (api *PublicSeeleAPI) GetBlockNumber(input interface{}, number *uint64) error {
	block, _ := api.s.chain.CurrentBlock()
	*number = block.Header.Height

	return nil
}

// GetBlockByNumber returns the requested block. When blockNr is -1 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlockByNumber(numberStr string, block *types.Block) error {
	store := api.s.chain.GetStore()
	if numberStr == "-1" {
		pBlock, _ := api.s.chain.CurrentBlock()
		*block = *pBlock
		return nil
	}
	number, err := strconv.ParseUint(numberStr, 10, 64)
	if err != nil {
		return err
	}
	hash, err := store.GetBlockHash(number)
	if err != nil {
		return err
	}
	pBlock, err := store.GetBlock(hash)
	if err != nil {
		return err
	}
	*block = *pBlock
	return nil
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned
func (api *PublicSeeleAPI) GetBlockByHash(hashHex string, block *types.Block) error {
	store := api.s.chain.GetStore()
	hashByte, err := hexutil.HexToBytes(hashHex)
	if err != nil {
		return err
	}
	hash := common.BytesToHash(hashByte)
	pBlock, err := store.GetBlock(hash)
	if err != nil {
		return err
	}
	*block = *pBlock
	return nil
}
