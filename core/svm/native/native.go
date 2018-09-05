package native

import (
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
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

// ProcessTransaction the system contract
func (n *NVM) ProcessTransaction(tx *types.Transaction) (*types.Receipt, error) {
	if n.contract == nil && system.GetContractByAddress(tx.Data.To) == nil {
		return nil, fmt.Errorf("use an invalid system contract")
	}

	usedGas := n.contract.RequiredGas(tx.Data.Payload)
	totalFee := new(big.Int).Add(usedGasFee(usedGas), tx.Data.Fee)

	receipt := &types.Receipt{
		UsedGas:         usedGas,
		TxHash:          tx.Hash,
		ContractAddress: tx.Data.To.Bytes(),
		TotalFee:        totalFee.Uint64(),
	}

	var err error
	ctx := system.NewContext(tx, n.statedb, n.blockHeader)
	if receipt.Result, err = n.contract.Run(tx.Data.Payload, ctx); err != nil {
		receipt.Result = []byte(err.Error())
		receipt.Failed = true
	}

	return receipt, nil
}

// usedGasFee returns the contract execution fee according to used gas.
func usedGasFee(usedGas uint64) *big.Int {
	return big.NewInt(0).SetUint64(usedGas)
}
