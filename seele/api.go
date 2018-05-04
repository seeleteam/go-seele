/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/p2p"
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
	balance := state.GetBalance(*account)
	result.Set(balance)
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
	*nonce = state.GetNonce(*account)

	return nil
}

// PublicNetworkAPI provides an API to access network information.
type PublicNetworkAPI struct {
	p2pServer      *p2p.Server
	networkVersion uint64
}

// NewPublicNetworkAPI creates a new PublicNetworkAPI object for rpc service.
func NewPublicNetworkAPI(p2pServer *p2p.Server, networkVersion uint64) *PublicNetworkAPI {
	return &PublicNetworkAPI{p2pServer, networkVersion}
}

// GetPeerCount returns the count of peers
func (n *PublicNetworkAPI) GetPeerCount(input interface{}, result *int) error {
	*result = n.p2pServer.PeerCount()
	return nil
}

// GetNetworkVersion returns the network version
func (n *PublicNetworkAPI) GetNetworkVersion(input interface{}, result *uint64) error {
	*result = n.networkVersion
	return nil
}

// RPCBlockInfo defines the block infos used by rpc
type RPCBlockInfo struct {
	HeadHash        common.Hash    `json:"headHash"`
	PreHash         common.Hash    `json:"preBlockHash"`
	Height          uint64         `json:"height"`
	Timestamp       *big.Int       `json:"timestamp"`
	Difficulty      *big.Int       `json:"difficulty"`
	TotleDifficulty *big.Int       `json:"totledifficulty"`
	Creator         common.Address `json:"creator"`
	Nonce           uint64         `json:"nonce"`
	TxCount         int            `json:"txcount"`
}

// GetBlockByHeight returns block infos by given height
func (api *PublicSeeleAPI) GetBlockByHeight(h uint64, ret *RPCBlockInfo) error {
	b, err := api.s.chain.GetBlockByHeight(h)
	if err == nil {
		// get totle difficulty
		td, err := api.s.chain.GetTdByHeight(h)
		if err != nil {
			return err
		}

		*ret = RPCBlockInfo{
			HeadHash:        b.HeaderHash,
			PreHash:         b.Header.PreviousBlockHash,
			Height:          b.Header.Height,
			Timestamp:       b.Header.CreateTimestamp,
			Difficulty:      b.Header.Difficulty,
			TotleDifficulty: td,
			Creator:         b.Header.Creator,
			Nonce:           b.Header.Nonce,
			TxCount:         len(b.Transactions),
		}
	}
	return err
}

// CurrentBlock return the best block of the blockchain
func (api *PublicSeeleAPI) CurrentBlock(arg interface{}, ret *RPCBlockInfo) error {
	curblock, _ := api.s.chain.CurrentBlock()
	// get totle difficulty
	td, err := api.s.chain.GetTdByHash(curblock.HeaderHash)
	if err != nil {
		return err
	}

	*ret = RPCBlockInfo{
		HeadHash:        curblock.HeaderHash,
		PreHash:         curblock.Header.PreviousBlockHash,
		Height:          curblock.Header.Height,
		Timestamp:       curblock.Header.CreateTimestamp,
		Difficulty:      curblock.Header.Difficulty,
		TotleDifficulty: td,
		Creator:         curblock.Header.Creator,
		Nonce:           curblock.Header.Nonce,
		TxCount:         len(curblock.Transactions),
	}

	return nil
}

// RPCBlockTransactions defines a struct that contains all transactions in the block
type RPCBlockTransactions struct {
	BlockHash common.Hash           `json:"blockHash"`
	Txs       []*RPCTransactionInfo `json:"transactions"`
}

// RPCTransactionInfo defines transaction info used by rpc
type RPCTransactionInfo struct {
	Hash         common.Hash     `json:"hash"`
	From         common.Address  `json:"from"`
	To           *common.Address `json:"to"`
	Amount       *big.Int        `json:"amount"`
	AccountNonce uint64          `json:"accountNonce"`
	Timestamp    uint64          `json:"time"`
}

// getTxs to get the transaction array
func getTxs(txs []*types.Transaction) []*RPCTransactionInfo {
	ret := make([]*RPCTransactionInfo, len(txs))
	for i, tx := range txs {
		ret[i] = &RPCTransactionInfo{
			Hash:         tx.Hash,
			From:         tx.Data.From,
			To:           tx.Data.To,
			Amount:       tx.Data.Amount,
			AccountNonce: tx.Data.AccountNonce,
			Timestamp:    tx.Data.Timestamp,
		}
	}
	return ret
}

// GetBlockTxsByHeight returns all transactions of a block
func (api *PublicSeeleAPI) GetBlockTxsByHeight(h uint64, ret *RPCBlockTransactions) error {
	b, err := api.s.chain.GetBlockByHeight(h)
	if err == nil {
		*ret = RPCBlockTransactions{
			BlockHash: b.HeaderHash,
			Txs:       getTxs(b.Transactions),
		}
	}
	return err
}
