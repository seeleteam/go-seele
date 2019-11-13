package core

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
)

const ()

// SubBlockChain we use the same struct as main chain
type SubBlockChain struct {
	sbc Blockchain
}

// since the subchain use the same structure as mainchain, here we just rewrite the methods from mainchain. (there is no sharding in subchain)
// things needed to change: rewardTx, Debt,
// functions: applyTxs / applyRewardAndRegularTxs /
func NewSubBlockChain(bcStore store.BlockchainStore, accountStateDB database.Database, recoveryPointFile string, engine consensus.Engine, startHeight int) (*Blockchain, error) {
	// subVerifier types.DebtVerifier // since subchain won't have sharding, so we just pass an empty subverifier
	return NewBlockchain(bcStore, accountStateDB, recoveryPointFile, engine, nil, startHeight)
}

func (subchain *SubBlockChain) AccountDB() database.Database {
	return subchain.sbc.AccountDB()
}

func (subchain *SubBlockChain) CurrentBlock() *types.Block {
	return subchain.sbc.currentBlock.Load().(*types.Block)
}

func (subchain *SubBlockChain) UpdateCurrentBlock(block *types.Block) {
	subchain.sbc.UpdateCurrentBlock(block)
}

func (subchain *SubBlockChain) AddBlockLeaves(blockIndex *BlockIndex) {
	subchain.sbc.AddBlockLeaves(blockIndex)
}

func (subchain *SubBlockChain) RemoveBlockLeaves(hash common.Hash) {
	subchain.sbc.RemoveBlockLeaves(hash)
}

func (subchain *SubBlockChain) CurrentHeader() *types.BlockHeader {
	return subchain.sbc.CurrentHeader()
}

func (subchain *SubBlockChain) GetCurrentState() (*state.Statedb, error) {
	return subchain.sbc.GetCurrentState()
}

func (subchain *SubBlockChain) GetHeaderByHeight(height uint64) *types.BlockHeader {
	return subchain.sbc.GetHeaderByHeight(height)
}

func (subchain *SubBlockChain) GetHeaderByHash(hash common.Hash) *types.BlockHeader {
	return subchain.sbc.GetHeaderByHash(hash)
}

func (subchain *SubBlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	return subchain.sbc.GetBlockByHash(hash)
}

func (subchain *SubBlockChain) GetState(root common.Hash) (*state.Statedb, error) {
	return subchain.sbc.GetState(root)
}

func (subchain *SubBlockChain) GetStateByRootAndBlockHash(root, blockHash common.Hash) (*state.Statedb, error) {
	return subchain.sbc.GetStateByRootAndBlockHash(root, blockHash)
}

func (subchain *SubBlockChain) Genesis() *types.Block {
	return subchain.sbc.Genesis()
}

func (subchain *SubBlockChain) GetCurrentInfo() (*types.Block, *state.Statedb, error) {
	return subchain.sbc.GetCurrentInfo()
}

func (subchain *SubBlockChain) WriteBlock(block *types.Block) error {
	return subchain.sbc.WriteBlock(block)
}

// func (subchain *SubBlockChain) WriteHeader(*types.BlockHeader) error {
// 	return subchain.sbc.WriteHeader()
// }

func (subchain *SubBlockChain) doWriteBlock(block *types.Block) error {
	return subchain.sbc.doWriteBlock(block)
}

func (subchain *SubBlockChain) validateBlock(block *types.Block) error {
	return subchain.sbc.validateBlock(block)
}

// // ValidateBlockHeader will call like core.ValidateBlockHeader(header, engine, bcsStore, chainReader)
// func (subchain *SubBlockChain) ValidateBlockHeader(header *types.BlockHeader, engine consensus.Engine, bcStore store.BlockchainStore, chainReader consensus.ChainReader) error{

// }

func (subchain *SubBlockChain) GetStore() store.BlockchainStore {
	return subchain.sbc.GetStore()
}

// todo
func (subchain *SubBlockChain) applyTxs(block *types.Block, root common.Hash) (*state.Statedb, []*types.Receipt, error) {
	return subchain.sbc.applyTxs(block, root)
}

// todo
func (subchain *SubBlockChain) applyRewardAndRegularTxs(statedb *state.Statedb, rewardTx *types.Transaction, regularTxs []*types.Transaction, blockHeader *types.BlockHeader) ([]*types.Receipt, error) {
	return subchain.sbc.applyRewardAndRegularTxs(statedb, rewardTx, regularTxs, blockHeader)
}

// todo
func (subchain *SubBlockChain) ApplyTransaction(tx *types.Transaction, txIndex int, coinbase common.Address, statedb *state.Statedb,
	blockHeader *types.BlockHeader) (*types.Receipt, error) {
	return subchain.sbc.ApplyTransaction(tx, txIndex, coinbase, statedb, blockHeader)
}

// todo
func (subchain *SubBlockChain) ApplyDebtWithoutVerify(statedb *state.Statedb, d *types.Debt, coinbase common.Address) error {
	return subchain.sbc.ApplyDebtWithoutVerify(statedb, d, coinbase)
}
