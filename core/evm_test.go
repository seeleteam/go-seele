/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/core/vm"
	"github.com/seeleteam/go-seele/crypto"
)

func Test_CreateContract(t *testing.T) {
	// Prepare blockchain store and state DB.
	db, dispose := newTestDatabase()
	defer dispose()
	bcStore := store.NewBlockchainDatabase(db)
	statedb, err := state.NewStatedb(common.EmptyHash, db)
	assert.Equal(t, err, nil)

	// Initialize account balance and nonce to create contract
	from := *crypto.MustGenerateRandomAddress()
	statedb.GetOrNewStateObject(from)
	statedb.SetBalance(from, big.NewInt(1000))
	statedb.SetNonce(from, 38)

	// Prepare tx to create contract: simple storage contract
	code, _ := hexutil.HexToBytes("608060405234801561001057600080fd5b50600560008190555060df806100276000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058207f6dc43a0d648e9f5a0cad5071cde46657de72eb87ab4cded53a7f1090f51e6d0029")
	tx, err := types.NewContractTransaction(from, big.NewInt(100), big.NewInt(2), 38, code)
	assert.Equal(t, err, nil)

	// Prepare block header
	header := &types.BlockHeader{
		PreviousBlockHash: crypto.MustHash("block previous hash"),
		Creator:           *crypto.MustGenerateRandomAddress(),
		StateHash:         crypto.MustHash("state root hash"),
		TxHash:            crypto.MustHash("tx root hash"),
		ReceiptHash:       crypto.MustHash("receipt root hash"),
		Difficulty:        big.NewInt(38),
		Height:            666,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		Nonce:             10,
		ExtraData:         make([]byte, 0),
	}

	// Prepare EVM and process tx
	evmContext := newEVMContext(tx, header, header.Creator, bcStore)
	receipt, err := processContract(evmContext, tx, 8, statedb, &vm.Config{})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(receipt.Result), 0)
	assert.Equal(t, receipt.TxHash, tx.CalculateHash())
	assert.Equal(t, receipt.ContractAddress, crypto.CreateAddress(from, 38))
}
