/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package evm

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
	"github.com/seeleteam/go-seele/database/leveldb"
)

//////////////////////////////////////////////////////////////////////////////////////////////////
// PLEASE USE REMIX (OR OTHER TOOLS) TO GENERATE CONTRACT CODE AND INPUT MESSAGE.
// Online: https://remix.ethereum.org/
// Github: https://github.com/ethereum/remix-ide
//////////////////////////////////////////////////////////////////////////////////////////////////

func mustHexToBytes(hex string) []byte {
	code, err := hexutil.HexToBytes(hex)
	if err != nil {
		panic(err)
	}

	return code
}

// preprocessContract creates the contract tx dependent state DB, blockchain store
// and a default account with specified balance and nonce.
func preprocessContract(balance, nonce uint64) (*state.Statedb, store.BlockchainStore, common.Address, func()) {
	db, dispose := leveldb.NewTestDatabase()

	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		dispose()
		panic(err)
	}

	// Create a default account to test contract.
	addr := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(addr)
	statedb.SetBalance(addr, new(big.Int).SetUint64(balance))
	statedb.SetNonce(addr, nonce)

	return statedb, store.NewBlockchainDatabase(db), addr, func() {
		dispose()
	}
}

func newTestBlockHeader() *types.BlockHeader {
	return &types.BlockHeader{
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
}

func Test_SimpleStorage(t *testing.T) {
	statedb, bcStore, address, dispose := preprocessContract(1000*common.SeeleToFan.Uint64(), 38)
	defer dispose()

	header := newTestBlockHeader()

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// Create contract tx, please refer to the code content in contract/solidity/simple_storage.sol
	code := mustHexToBytes("0x608060405234801561001057600080fd5b50600560008190555060df806100276000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058207f6dc43a0d648e9f5a0cad5071cde46657de72eb87ab4cded53a7f1090f51e6d0029")
	createContractTx, err := types.NewContractTransaction(address, new(big.Int), big.NewInt(1), 38, code)
	assert.Equal(t, err, nil)

	svm := &SVM{Evm: NewEVMByDefaultConfig(createContractTx, statedb, header, bcStore)}
	receipt, err := svm.Process(createContractTx, 8)
	// Validate receipt of contract creation.
	assert.Equal(t, err, nil)
	assert.Equal(t, receipt.TxHash, createContractTx.CalculateHash())
	assert.Equal(t, receipt.ContractAddress, crypto.CreateAddress(address, 38).Bytes())
	assert.Equal(t, receipt.UsedGas, uint64(64707))
	assert.Equal(t, receipt.TotalFee, (big.NewInt(0).Add(createContractTx.Data.Fee, contractFeeSimple)).Uint64())

	// Validate the state DB after contract created.
	contractAddr := common.BytesToAddress(receipt.ContractAddress)
	assert.Equal(t, statedb.Exist(contractAddr), true)
	assert.Equal(t, statedb.GetBalance(contractAddr), new(big.Int))
	assert.Equal(t, statedb.GetNonce(contractAddr), uint64(1))

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// Call contract tx: SimpleStorage.get(), it returns 5 as initialized in constructor.
	input := mustHexToBytes("0x6d4ce63c")
	callContractTx, err := types.NewMessageTransaction(address, contractAddr, new(big.Int), big.NewInt(1), 1, input)
	assert.Equal(t, err, nil)

	svm = &SVM{Evm: NewEVMByDefaultConfig(callContractTx, statedb, header, bcStore)}
	receipt, err = svm.Process(callContractTx, 9)

	// Validate receipt of contract call
	assert.Equal(t, err, nil)
	assert.Equal(t, receipt.Result, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5})
	assert.Equal(t, receipt.TxHash, callContractTx.CalculateHash())
	assert.Equal(t, len(receipt.ContractAddress), 0)
	assert.Equal(t, receipt.UsedGas, uint64(424))
	assert.Equal(t, receipt.TotalFee, (big.NewInt(0).Add(createContractTx.Data.Fee, usedGasFee(receipt.UsedGas))).Uint64())

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// Call contract tx: SimpleStorage.set(666)
	input = mustHexToBytes("0x60fe47b1000000000000000000000000000000000000000000000000000000000000029a")
	callContractTx, err = types.NewMessageTransaction(address, contractAddr, new(big.Int), big.NewInt(1), 1, input)
	assert.Equal(t, err, nil)

	svm = &SVM{Evm: NewEVMByDefaultConfig(callContractTx, statedb, header, bcStore)}
	receipt, err = svm.Process(callContractTx, 10)

	// Validate receipt contract call
	assert.Equal(t, err, nil)
	assert.Equal(t, len(receipt.Result), 0)
	assert.Equal(t, receipt.TxHash, callContractTx.CalculateHash())
	assert.Equal(t, len(receipt.ContractAddress), 0)
	assert.Equal(t, receipt.UsedGas, uint64(5205))
	assert.Equal(t, receipt.TotalFee, (big.NewInt(0).Add(createContractTx.Data.Fee, usedGasFee(receipt.UsedGas))).Uint64())

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// Call contract tx: SimpleStorage.get(), it returns 666 as set above.
	input = mustHexToBytes("0x6d4ce63c")
	callContractTx, err = types.NewMessageTransaction(address, contractAddr, new(big.Int), big.NewInt(1), 1, input)
	assert.Equal(t, err, nil)

	svm = &SVM{Evm: NewEVMByDefaultConfig(callContractTx, statedb, header, bcStore)}
	receipt, err = svm.Process(callContractTx, 11)

	// Validate receipt of contract call
	assert.Equal(t, err, nil)
	assert.Equal(t, receipt.Result, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 154}) // 666
	assert.Equal(t, receipt.TxHash, callContractTx.CalculateHash())
	assert.Equal(t, len(receipt.ContractAddress), 0)
	assert.Equal(t, receipt.UsedGas, uint64(424))
	assert.Equal(t, receipt.TotalFee, (big.NewInt(0).Add(createContractTx.Data.Fee, usedGasFee(receipt.UsedGas))).Uint64())
}

func Test_InsufficientBalance(t *testing.T) {
	// Account balance: 1 seele
	statedb, bcStore, address, dispose := preprocessContract(common.SeeleToFan.Uint64(), 38)
	defer dispose()

	header := newTestBlockHeader()

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// Create contract tx, please refer to the code content in contract/solidity/simple_storage.sol
	code := mustHexToBytes("0x608060405234801561001057600080fd5b50600560008190555060df806100276000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058207f6dc43a0d648e9f5a0cad5071cde46657de72eb87ab4cded53a7f1090f51e6d0029")
	// tx fee: 1 seele
	createContractTx, err := types.NewContractTransaction(address, new(big.Int), common.SeeleToFan, 38, code)
	assert.Equal(t, err, nil)

	svm := &SVM{Evm: NewEVMByDefaultConfig(createContractTx, statedb, header, bcStore)}
	_, err = svm.Process(createContractTx, 8)
	assert.Equal(t, err, vm.ErrInsufficientBalance)
}

func Benchmark_CreateContract(b *testing.B) {
	statedb, bcStore, address, dispose := preprocessContract(common.SeeleToFan.Uint64(), 38)
	defer dispose()

	statedb.SetBalance(address, new(big.Int).Mul(common.SeeleToFan, big.NewInt(100000000)))
	header := newTestBlockHeader()

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// Create contract tx, please refer to the code content in contract/solidity/simple_storage.sol
	code := mustHexToBytes("0x608060405234801561001057600080fd5b50600560008190555060df806100276000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058207f6dc43a0d648e9f5a0cad5071cde46657de72eb87ab4cded53a7f1090f51e6d0029")
	amount := big.NewInt(0)
	fee := big.NewInt(1)
	nonce := uint64(38)
	createContractTx, _ := types.NewContractTransaction(address, amount, fee, nonce, code)

	svm := &SVM{Evm: NewEVMByDefaultConfig(createContractTx, statedb, header, bcStore)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svm.Process(createContractTx, 8)
		createContractTx.Data.AccountNonce++
	}
}

func Benchmark_CallContract(b *testing.B) {
	statedb, bcStore, address, dispose := preprocessContract(common.SeeleToFan.Uint64(), 38)
	defer dispose()

	statedb.SetBalance(address, new(big.Int).Mul(common.SeeleToFan, big.NewInt(100000000)))
	header := newTestBlockHeader()

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// Create contract tx, please refer to the code content in contract/solidity/simple_storage.sol
	code := mustHexToBytes("0x608060405234801561001057600080fd5b50600560008190555060df806100276000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058207f6dc43a0d648e9f5a0cad5071cde46657de72eb87ab4cded53a7f1090f51e6d0029")
	amount := big.NewInt(0)
	fee := big.NewInt(1)
	nonce := uint64(38)
	createContractTx, _ := types.NewContractTransaction(address, amount, fee, nonce, code)

	svm := &SVM{Evm: NewEVMByDefaultConfig(createContractTx, statedb, header, bcStore)}
	receipt, _ := svm.Process(createContractTx, 8)
	contractAddr := common.BytesToAddress(receipt.ContractAddress)

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// Call contract tx: SimpleStorage.get(), it returns 5 as initialized in constructor.
	input := mustHexToBytes("0x6d4ce63c")
	nonce = uint64(1)
	callContractTx, _ := types.NewMessageTransaction(address, contractAddr, amount, fee, nonce, input)

	svm = &SVM{Evm: NewEVMByDefaultConfig(callContractTx, statedb, header, bcStore)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svm.Process(callContractTx, 9)
	}
}
