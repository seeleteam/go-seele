/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package seele

import (
	"bytes"
	"context"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	api2 "github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/consensus/factory"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/stretchr/testify/assert"
)

func Test_PublicSeeleAPI(t *testing.T) {
	conf := getTmpConfig()
	serviceContext := ServiceContext{
		DataDir: filepath.Join(common.GetTempFolder(), ".PublicSeeleAPI"),
	}

	var key interface{} = "ServiceContext"
	ctx := context.WithValue(context.Background(), key, serviceContext)
	dataDir := ctx.Value("ServiceContext").(ServiceContext).DataDir
	log := log.GetLogger("seele")
	ss, err := NewSeeleService(ctx, conf, log, factory.MustGetConsensusEngine(common.Sha256Algorithm), nil)
	if err != nil {
		t.Fatal()
	}

	api := NewPublicSeeleAPI(ss)
	defer func() {
		api.s.Stop()
		os.RemoveAll(dataDir)
	}()
	var info api2.GetMinerInfo
	info, err = api.GetInfo()
	assert.Equal(t, err, nil)
	if !bytes.Equal(conf.SeeleConfig.Coinbase[0:], info.Coinbase[0:]) {
		t.Fail()
	}
}

func Test_GetLogs(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".GetLogs")
	api := newTestAPI(t, dbPath)
	defer func() {
		api.s.Stop()
		os.RemoveAll(dbPath)
	}()

	// Create a simple_storage_1 contract
	bytecode, _ := hexutil.HexToBytes("0x608060405234801561001057600080fd5b506005600055610141806100256000396000f30060806040526004361061004b5763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166360fe47b181146100505780636d4ce63c1461006a575b600080fd5b34801561005c57600080fd5b50610068600435610091565b005b34801561007657600080fd5b5061007f610096565b60408051918252519081900360200190f35b600055565b60408051600181526002602082015281516000927f672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642928290030190a160408051600381526004602082015281517f1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5929181900390910190a150600054905600a165627a7a72305820da608eada1eb6f77ba481c426f9c58dedad4df982b20f3f62efac1dbb710a7cc0029")
	statedb, _ := api.s.chain.GetCurrentState()
	from := getFromAddress(statedb)
	createContractTx, _ := types.NewContractTransaction(from, big.NewInt(0), big.NewInt(1), 500000, 0, bytecode)
	contractAddressByte := sendTx(t, api, statedb, createContractTx)

	// Get the contract address
	contractAddressHex := hexutil.BytesToHex(contractAddressByte)
	contractAddress, err := common.HexToAddress(contractAddressHex)
	assert.Equal(t, err, nil)

	// The origin statedb
	statedbOri, err := api.s.chain.GetCurrentState()
	assert.Equal(t, err, nil)

	// Call the get function
	msg, err := hexutil.HexToBytes("0x6d4ce63c")
	assert.Equal(t, err, nil)
	getTx, err := types.NewMessageTransaction(from, contractAddress, big.NewInt(0), big.NewInt(10), 500000, 1, msg)
	assert.Equal(t, err, nil)
	receipt, err := api.s.chain.ApplyTransaction(getTx, 0, api.s.miner.GetCoinbase(), statedbOri, api.s.chain.CurrentBlock().Header)
	assert.Equal(t, err, nil)

	// Save the statedb and receipts
	batch := api.s.accountStateDB.NewBatch()
	block := api.s.chain.CurrentBlock()
	block.Header.StateHash, _ = statedb.Commit(batch)
	batch.Commit()
	api.s.chain.GetStore().PutReceipts(block.HeaderHash, []*types.Receipt{receipt})

	abifile := `[
	{ "constant" : false, "inputs": [ { "name": "x", "type": "uint256" } ], "name": "set", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "constant" : false, "inputs": [], "name": "get", "outputs": [ { "name": "", "type": "uint256" } ], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "inputs": [], "payable": false, "stateMutability": "nonpayable", "type": "constructor" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "uint256" }, { "indexed": false, "name": "", "type": "uint256" } ], "name": "getX", "type": "event" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "uint256" }, { "indexed": false, "name": "", "type": "uint256" } ], "name": "getY", "type": "event" }
]`
	// Verify the result
	result, err := api.GetLogs(-1, contractAddress, abifile, "getX")
	assert.Equal(t, err, nil)
	assert.Equal(t, len(result), 1)
	assert.Equal(t, result[0].Txhash, receipt.TxHash)
	addr := result[0].Address
	assert.Equal(t, addr, contractAddress)

	// Verify the error contractAddress and error event
	// error contract address is used to compare with the log, so if there is no relevant log, GetLogs returns empty slice with no error.
	result, err = api.GetLogs(-1, common.EmptyAddress, abifile, "getX")
	assert.NoError(t, err)
	assert.Equal(t, len(result), 0)
	result, err = api.GetLogs(-1, contractAddress, abifile, "get")
	assert.Error(t, err)
}

func newTestAPI(t *testing.T, dbPath string) *PublicSeeleAPI {
	conf := getTmpConfig()
	serviceContext := ServiceContext{
		DataDir: dbPath,
	}
	var key interface{} = "ServiceContext"
	ctx := context.WithValue(context.Background(), key, serviceContext)
	log := log.GetLogger("seele")
	ss, err := NewSeeleService(ctx, conf, log, factory.MustGetConsensusEngine(common.Sha256Algorithm), nil)
	assert.Equal(t, err, nil)
	return NewPublicSeeleAPI(ss)
}

func sendTx(t *testing.T, api *PublicSeeleAPI, statedb *state.Statedb, tx *types.Transaction) []byte {
	receipt, err := api.s.chain.ApplyTransaction(tx, 0, api.s.miner.GetCoinbase(), statedb, api.s.chain.CurrentBlock().Header)
	assert.Equal(t, err, nil)
	assert.Equal(t, receipt.Failed, false)

	// Save the statedb
	batch := api.s.accountStateDB.NewBatch()
	block := api.s.chain.CurrentBlock()
	block.Header.StateHash, _ = statedb.Commit(batch)
	block.Header.Height++
	block.Header.PreviousBlockHash = block.HeaderHash
	block.HeaderHash = block.Header.Hash()
	api.s.chain.GetStore().PutBlock(block, big.NewInt(1), true)
	batch.Commit()
	return receipt.ContractAddress
}

func getFromAddress(statedb *state.Statedb) common.Address {
	from := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(from)
	statedb.SetBalance(from, common.SeeleToFan)
	statedb.SetNonce(from, 0)
	return from
}

func Test_Call(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".Call")
	api := newTestAPI(t, dbPath)
	defer func() {
		api.s.Stop()
		os.RemoveAll(dbPath)
	}()

	// Create a contract/solidity/simple_storage.sol contract, get = 5
	bytecode, _ := hexutil.HexToBytes("0x608060405234801561001057600080fd5b50600560008190555060df806100276000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058207f6dc43a0d648e9f5a0cad5071cde46657de72eb87ab4cded53a7f1090f51e6d0029")
	statedb, _ := api.s.chain.GetCurrentState()
	from := getFromAddress(statedb)
	createContractTx, _ := types.NewContractTransaction(from, big.NewInt(0), big.NewInt(1), 500000, 0, bytecode)
	contractAddressByte := sendTx(t, api, statedb, createContractTx)

	// Get the contract address
	contractAddressHex := hexutil.BytesToHex(contractAddressByte)
	contractAddress, err := common.HexToAddress(contractAddressHex)
	assert.Equal(t, err, nil)

	// The origin statedb
	statedbOri, err := api.s.chain.GetCurrentState()
	assert.Equal(t, err, nil)

	// get payload
	payload := "0x6d4ce63c"

	// Verify the result = 5
	result := make(map[string]interface{})
	result, err = api.Call(contractAddress.Hex(), payload, -1)
	assert.Equal(t, err, nil)
	assert.Equal(t, result["result"], "0x0000000000000000000000000000000000000000000000000000000000000005")

	// It is no diffrence to the origin statedb
	statedbCur, err := api.s.chain.GetCurrentState()
	assert.Equal(t, err, nil)
	assert.Equal(t, statedbOri, statedbCur)

	// set 23 payload
	bytecode, _ = hexutil.HexToBytes("0x60fe47b10000000000000000000000000000000000000000000000000000000000000017")
	callContractTx, _ := types.NewMessageTransaction(from, contractAddress, big.NewInt(0), big.NewInt(1), 500000, 1, bytecode)
	_ = sendTx(t, api, statedbCur, callContractTx)

	// Verify the result = 23
	result, err = api.Call(contractAddress.Hex(), payload, -1)
	assert.Equal(t, err, nil)
	assert.Equal(t, result["result"], "0x0000000000000000000000000000000000000000000000000000000000000017")

	// Verify the history result = 5
	height, err := api2.NewPublicSeeleAPI(NewSeeleBackend(api.s)).GetBlockHeight()
	assert.Equal(t, err, nil)
	result, err = api.Call(contractAddress.Hex(), payload, int64(height-1))
	assert.Equal(t, err, nil)
	assert.Equal(t, result["result"], "0x0000000000000000000000000000000000000000000000000000000000000005")

	// Verify the invalid contractAddress and payload
	result, err = api.Call("contractAddress.Hex()", payload, -1)
	assert.Equal(t, err == nil, false)
	result, err = api.Call(contractAddress.Hex(), "payload", -1)
	assert.Equal(t, err == nil, false)
}

func Test_EstimateGas(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".EstimateGas")
	api := newTestAPI(t, dbPath)
	defer func() {
		api.s.Stop()
		os.RemoveAll(dbPath)
	}()

	statedb, _ := api.s.chain.GetCurrentState()
	from := getFromAddress(statedb)
	// Save the statedb
	batch := api.s.accountStateDB.NewBatch()
	block := api.s.chain.CurrentBlock()
	block.Header.StateHash, _ = statedb.Commit(batch)
	block.Header.Height++
	block.Header.PreviousBlockHash = block.HeaderHash
	block.HeaderHash = block.Header.Hash()
	api.s.chain.GetStore().PutBlock(block, big.NewInt(1), true)
	batch.Commit()

	// Transfer - common shard
	to1 := crypto.MustGenerateShardAddress(from.Shard())
	transferCSTx, err1 := types.NewTransaction(from, *to1, big.NewInt(1), big.NewInt(1), statedb.GetNonce(from))
	assert.NoError(t, err1)
	estimateGas1, err2 := api.EstimateGas(transferCSTx)
	assert.NoError(t, err2)
	assert.Equal(t, estimateGas1, types.TransferAmountIntrinsicGas)

	// Transfer - different shard
	to2 := crypto.MustGenerateRandomAddress()
	for to2.Shard() == from.Shard() {
		to2 = crypto.MustGenerateRandomAddress()
	}
	transferDSTx, err3 := types.NewTransaction(from, *to2, big.NewInt(1), big.NewInt(1), statedb.GetNonce(from))
	assert.NoError(t, err3)
	estimateGas2, err4 := api.EstimateGas(transferDSTx)
	assert.NoError(t, err4)
	assert.Equal(t, estimateGas2, types.TransferAmountIntrinsicGas*2)

	// Create a contract/solidity/simple_storage.sol contract, get = 5
	bytecode, err5 := hexutil.HexToBytes("0x608060405234801561001057600080fd5b50600560008190555060df806100276000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058207f6dc43a0d648e9f5a0cad5071cde46657de72eb87ab4cded53a7f1090f51e6d0029")
	assert.NoError(t, err5)
	createContractTx, err6 := types.NewContractTransaction(from, big.NewInt(0), big.NewInt(1), 500000, 0, bytecode)
	assert.NoError(t, err6)
	estimateGas3, err7 := api.EstimateGas(createContractTx)
	assert.NoError(t, err7)
	assert.NotZero(t, estimateGas3)

	// Call contract
	bytecode1, err8 := hexutil.HexToBytes("0x6d4ce63c")
	assert.NoError(t, err8)
	callContractTx, err9 := types.NewMessageTransaction(from, createContractTx.Data.To, big.NewInt(0), big.NewInt(1), 500000, 0, bytecode1)
	assert.NoError(t, err9)
	estimateGas4, err10 := api.EstimateGas(callContractTx)
	assert.NoError(t, err10)
	assert.NotZero(t, estimateGas4)
}

func Test_GetInfo(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".GetLogs")
	api := newTestAPI(t, dbPath)
	defer func() {
		api.s.Stop()
		os.RemoveAll(dbPath)
	}()

	info, err := api.GetInfo()
	assert.Nil(t, err)
	assert.NotNil(t, info)
	assert.NotNil(t, info.Coinbase)
	assert.Equal(t, info.CurrentBlockHeight, uint64(0))
	assert.Equal(t, info.Shard, uint(0))
	assert.Equal(t, info.MinerStatus, "Stopped")
	assert.Equal(t, info.HeaderHash.Hex(), "0xb5a0c3f0d36ce6dc05f97ba393a43505055b5ab7b9d5240c5f37e37b778634de")
}
