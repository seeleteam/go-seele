/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/params"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/core/vm"
)

// newEVMContext creates a new context for use in the EVM.
func newEVMContext(tx *types.Transaction, header *types.BlockHeader, minerAddress common.Address, bcStore store.BlockchainStore) *vm.Context {
	canTransferFunc := func(db vm.StateDB, addr common.Address, amount *big.Int) bool {
		return db.GetBalance(addr).Cmp(amount) >= 0
	}

	transferFunc := func(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
		db.SubBalance(sender, amount)
		db.AddBalance(recipient, amount)
	}

	heightToHashMapping := map[uint64]common.Hash{
		header.Height - 1: header.PreviousBlockHash,
	}
	getHashFunc := func(height uint64) (common.Hash, error) {
		for preHash := header.PreviousBlockHash; ; {
			if hash, ok := heightToHashMapping[height]; ok {
				return hash, nil
			}

			preHeader, err := bcStore.GetBlockHeader(preHash)
			if err != nil {
				return common.EmptyHash, err
			}

			heightToHashMapping[preHeader.Height-1] = preHeader.PreviousBlockHash
			preHash = preHeader.PreviousBlockHash
		}
	}

	return &vm.Context{
		CanTransfer: canTransferFunc,
		Transfer:    transferFunc,
		GetHash:     getHashFunc,
		Origin:      tx.Data.From,
		Coinbase:    minerAddress,
		BlockNumber: new(big.Int).SetUint64(header.Height),
		Time:        new(big.Int).Set(header.CreateTimestamp),
		Difficulty:  new(big.Int).Set(header.Difficulty),
		// GasLimit:    header.GasLimit,
		// GasPrice:    new(big.Int).Set(tx.GasPrice()),
	}
}

// processContract process the specified contract tx and return the receipt.
func processContract(context *vm.Context, tx *types.Transaction, txIndex int, statedb *state.Statedb, vmConfig *vm.Config) (*types.Receipt, error) {
	statedb.Prepare(txIndex)
	evm := vm.NewEVM(*context, statedb, getDefaultChainConfig(), *vmConfig)

	var err error
	caller := vm.AccountRef(tx.Data.From)
	receipt := &types.Receipt{TxHash: tx.Hash}

	// Currently, use math.MaxUint64 gas to bypass ErrInsufficientBalance error.
	if tx.Data.To == nil {
		receipt.Result, receipt.ContractAddress, _, err = evm.Create(caller, tx.Data.Payload, math.MaxUint64, tx.Data.Amount)
	} else {
		statedb.SetNonce(tx.Data.From, statedb.GetNonce(tx.Data.From)+1)
		receipt.Result, _, err = evm.Call(caller, *tx.Data.To, tx.Data.Payload, math.MaxUint64, tx.Data.Amount)
	}

	if err != nil {
		return nil, err
	}

	receipt.PostState = statedb.Commit(nil)
	receipt.Logs = statedb.GetCurrentLogs()
	if receipt.Logs == nil {
		receipt.Logs = make([]*types.Log, 0)
	}

	return receipt, nil
}

func getDefaultChainConfig() *params.ChainConfig {
	return &params.ChainConfig{
		ChainId:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: nil,
		Ethash:              new(params.EthashConfig),
	}
}
