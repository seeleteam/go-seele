package native

import (
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/core/vm"
)

// NVM implemented svm by system contract
type NVM struct {
	tx          *types.Transaction
	statedb     *state.Statedb
	blockHeader *types.BlockHeader
	bcStore     store.BlockchainStore
	contract    system.Contract
}

// NewNativeVM is to process the system contract, You must guarantee that the contract is non-empty
func NewNativeVM(tx *types.Transaction, statedb *state.Statedb, blockHeader *types.BlockHeader, bcStore store.BlockchainStore, contract system.Contract) *NVM {
	return &NVM{tx, statedb, blockHeader, bcStore, contract}
}

// Process the system contract
func (n *NVM) Process(tx *types.Transaction, txIndex int) (*types.Receipt, error) {
	if n.contract == nil && system.GetContractByAddress(tx.Data.To) == nil {
		return nil, fmt.Errorf("use an invalid system contract")
	}

	// TODO separate the commom statedb operation
	n.statedb.Prepare(txIndex)

	usedGas := n.contract.RequiredGas(tx.Data.Payload)
	totalFee := new(big.Int).Add(usedGasFee(usedGas), tx.Data.Fee)
	if balance := n.statedb.GetBalance(tx.Data.From); balance.Cmp(totalFee) < 0 {
		return nil, vm.ErrInsufficientBalance
	}

	receipt := &types.Receipt{
		UsedGas:         usedGas,
		TxHash:          tx.Hash,
		ContractAddress: tx.Data.To.Bytes(),
		TotalFee:        totalFee.Uint64(),
	}

	// add from nonce
	n.statedb.SetNonce(tx.Data.From, tx.Data.AccountNonce+1)

	var err error
	ctx := system.NewContext(tx, n.statedb)
	if receipt.Result, err = n.contract.Run(tx.Data.Payload, ctx); err != nil {
		receipt.Result = []byte(err.Error())
		receipt.Failed = true
	}

	// transfer fee to coinbase
	n.statedb.SubBalance(tx.Data.From, totalFee)
	n.statedb.AddBalance(n.blockHeader.Creator, totalFee)

	if receipt.PostState, err = n.statedb.Hash(); err != nil {
		return nil, err
	}

	receipt.Logs = n.statedb.GetCurrentLogs()
	if receipt.Logs == nil {
		receipt.Logs = make([]*types.Log, 0)
	}

	return receipt, nil
}

// usedGasFee returns the contract execution fee according to used gas.
func usedGasFee(usedGas uint64) *big.Int {
	return big.NewInt(0).SetUint64(usedGas)
}
