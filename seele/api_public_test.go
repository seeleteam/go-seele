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

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/node"
	"github.com/stretchr/testify/assert"
)

func getTmpConfig() *node.Config {
	acctAddr := crypto.MustGenerateRandomAddress()

	return &node.Config{
		SeeleConfig: node.SeeleConfig{
			TxConf:   *core.DefaultTxPoolConfig(),
			Coinbase: *acctAddr,
		},
	}
}

func Test_PublicSeeleAPI(t *testing.T) {
	conf := getTmpConfig()
	serviceContext := ServiceContext{
		DataDir: common.GetTempFolder(),
	}

	var key interface{} = "ServiceContext"
	ctx := context.WithValue(context.Background(), key, serviceContext)
	dataDir := ctx.Value("ServiceContext").(ServiceContext).DataDir
	defer os.RemoveAll(dataDir)
	log := log.GetLogger("seele")
	ss, err := NewSeeleService(ctx, conf, log)
	if err != nil {
		t.Fatal()
	}

	api := NewPublicSeeleAPI(ss)
	var info MinerInfo
	info, err = api.GetInfo()
	assert.Equal(t, err, nil)

	if !bytes.Equal(conf.SeeleConfig.Coinbase[0:], info.Coinbase[0:]) {
		t.Fail()
	}
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
	createContractTx, _ := types.NewContractTransaction(from, big.NewInt(0), big.NewInt(1), 0, bytecode)
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
	result, err = api.Call(contractAddress.ToHex(), payload, -1)
	assert.Equal(t, err, nil)
	assert.Equal(t, result["result"], "0x0000000000000000000000000000000000000000000000000000000000000005")

	// It is no diffrence to the origin statedb
	statedbCur, err := api.s.chain.GetCurrentState()
	assert.Equal(t, err, nil)
	assert.Equal(t, statedbOri, statedbCur)

	// set 23 payload
	bytecode, _ = hexutil.HexToBytes("0x60fe47b10000000000000000000000000000000000000000000000000000000000000017")
	callContractTx, _ := types.NewMessageTransaction(from, contractAddress, big.NewInt(0), big.NewInt(1), 0, bytecode)
	_ = sendTx(t, api, statedbCur, callContractTx)

	// Verify the result = 23
	result, err = api.Call(contractAddress.ToHex(), payload, -1)
	assert.Equal(t, err, nil)
	assert.Equal(t, result["result"], "0x0000000000000000000000000000000000000000000000000000000000000017")

	// Verify the history result = 5
	height, err := api.GetBlockHeight()
	assert.Equal(t, err, nil)
	result, err = api.Call(contractAddress.ToHex(), payload, int64(height-1))
	assert.Equal(t, err, nil)
	assert.Equal(t, result["result"], "0x0000000000000000000000000000000000000000000000000000000000000005")

	// Verify the invalid contractAddress and payload
	result, err = api.Call("contractAddress.ToHex()", payload, -1)
	assert.Equal(t, err == nil, false)
	result, err = api.Call(contractAddress.ToHex(), "payload", -1)
	assert.Equal(t, err == nil, false)
}

func Test_GetLogs(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".GetLogs")

	api := newTestAPI(t, dbPath)
	defer func() {
		api.s.Stop()
		os.RemoveAll(dbPath)
	}()

	// Create a simple_storage_1 contract
	bytecode, _ := hexutil.HexToBytes("0x6080604052601760005534801561001557600080fd5b5061025f806100256000396000f30060806040526004361061004c576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b1146100515780636d4ce63c1461007e575b600080fd5b34801561005d57600080fd5b5061007c600480360381019080803590602001909291905050506100a9565b005b34801561008a57600080fd5b5061009361011b565b6040518082815260200191505060405180910390f35b7fe84bb31d4e9adbff26e80edeecb6cf8f3a95d1ba519cf60a08a6e6f8d62d81006040518080602001828103825260078152602001807f6765744c6f67320000000000000000000000000000000000000000000000000081525060200191505060405180910390a18060008190555050565b60007f978acaf30839c63aff19afed19ff8f3a430103773a67e3890aa1639af9a71bc433604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200180602001828103825260068152602001807f6765744c6f6700000000000000000000000000000000000000000000000000008152506020019250505060405180910390a17f523b2fb716b59c8e374bb3ea0f14ce672f9ac295b25470c403ad377306abb1026040518080602001828103825260078152602001807f6765744c6f67310000000000000000000000000000000000000000000000000081525060200191505060405180910390a161022b60106100a9565b6000549050905600a165627a7a72305820e12478ad92a5a4181935da97e24de739c4928ac47b2c5c1cd3423513298c62390029")
	statedb, _ := api.s.chain.GetCurrentState()
	from := getFromAddress(statedb)
	createContractTx, _ := types.NewContractTransaction(from, big.NewInt(0), big.NewInt(1), 0, bytecode)
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
	getTx, err := types.NewMessageTransaction(from, contractAddress, big.NewInt(0), big.NewInt(10), 1, msg)
	assert.Equal(t, err, nil)

	receipt, err := api.s.chain.ApplyTransaction(getTx, 0, api.s.miner.GetCoinbase(), statedbOri, api.s.chain.CurrentBlock().Header)
	assert.Equal(t, err, nil)

	// Save the statedb and receipts
	batch := api.s.accountStateDB.NewBatch()
	block := api.s.chain.CurrentBlock()
	block.Header.StateHash, _ = statedb.Commit(batch)
	batch.Commit()
	api.s.chain.GetStore().PutReceipts(block.HeaderHash, []*types.Receipt{receipt})

	// Verify the result
	payload := "0xe84bb31d4e9adbff26e80edeecb6cf8f3a95d1ba519cf60a08a6e6f8d62d8100"
	result, err := api.GetLogs(-1, contractAddress.ToHex(), payload)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(result), 1)
	assert.Equal(t, result[0].Txhash, receipt.TxHash)

	addr := result[0].Log.Address
	assert.Equal(t, addr, contractAddress)

	name := result[0].Log.Topics
	assert.Equal(t, name[0].ToHex(), payload)

	// Verify the invalid contractAddress and payload
	result, err = api.GetLogs(-1, "contractAddress.ToHex()", payload)
	assert.Equal(t, err == nil, false)
	result, err = api.GetLogs(-1, contractAddress.ToHex(), "payload")
	assert.Equal(t, err == nil, false)
}

func newTestAPI(t *testing.T, dbPath string) *PublicSeeleAPI {
	conf := getTmpConfig()
	serviceContext := ServiceContext{
		DataDir: dbPath,
	}

	var key interface{} = "ServiceContext"
	ctx := context.WithValue(context.Background(), key, serviceContext)

	log := log.GetLogger("seele")
	ss, err := NewSeeleService(ctx, conf, log)
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

func callContract(t *testing.T, api *PublicSeeleAPI, contractAddress *common.Address, payload string) {
	// Construct a create contract transaction
	bytecode, err := hexutil.HexToBytes(payload)
	assert.Equal(t, err, nil)

	statedb, err := api.s.chain.GetCurrentState()
	assert.Equal(t, err, nil)

	from := getFromAddress(statedb)
	callContractTx, err := types.NewMessageTransaction(from, *contractAddress, big.NewInt(0), big.NewInt(1), 0, bytecode)
	assert.Equal(t, err, nil)

	receipt, err := api.s.chain.ApplyTransaction(callContractTx, 0, api.s.miner.GetCoinbase(), statedb, api.s.chain.CurrentBlock().Header)
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
}

func getFromAddress(statedb *state.Statedb) common.Address {
	from := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(from)
	statedb.SetBalance(from, common.SeeleToFan)
	statedb.SetNonce(from, 0)
	return from
}

func Test_GetBlocks(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".GetBlocks")

	api := newTestAPI(t, dbPath)
	defer func() {
		api.s.Stop()
		os.RemoveAll(dbPath)
	}()

	block0 := newTestBlock(0)
	err := api.s.chain.GetStore().PutBlock(block0, block0.Header.Difficulty, true)
	assert.Equal(t, err, nil)
	block1 := newTestBlock(1)
	err = api.s.chain.GetStore().PutBlock(block1, block1.Header.Difficulty, true)
	assert.Equal(t, err, nil)
	block2 := newTestBlock(2)
	err = api.s.chain.GetStore().PutBlock(block2, block2.Header.Difficulty, true)
	assert.Equal(t, err, nil)

	result, err := api.GetBlocks(2, true, 2)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(result), 2)
	assert.Equal(t, result[0]["hash"].(string), block2.Header.Hash().ToHex())
	assert.Equal(t, result[1]["hash"].(string), block1.Header.Hash().ToHex())

	result, err = api.GetBlocks(-1, true, 1)
	assert.Equal(t, err == nil, true)
	assert.Equal(t, len(result), 1)

	result, err = api.GetBlocks(2, true, 6)
	assert.Equal(t, err == nil, true)
	assert.Equal(t, len(result), 3)

	result, err = api.GetBlocks(2, true, 600)
	assert.Equal(t, err == nil, true)
	assert.Equal(t, len(result), 3)

	result, err = api.GetBlocks(2, true, 0)
	assert.Equal(t, err == nil, true)
	assert.Equal(t, len(result), 0)

	result, err = api.GetBlocks(4, true, 2)
	assert.Equal(t, err != nil, true)
	assert.Equal(t, len(result), 0)

	result, err = api.GetBlocks(2, true, 3)
	assert.Equal(t, err == nil, true)
	assert.Equal(t, len(result), 3)
}

func newTestBlock(height uint64) *types.Block {
	header := &types.BlockHeader{
		PreviousBlockHash: common.StringToHash("PreviousBlockHash"),
		Creator:           *crypto.MustGenerateRandomAddress(),
		StateHash:         common.StringToHash("StateHash"),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        big.NewInt(1),
		Height:            height,
		CreateTimestamp:   big.NewInt(1),
		Nonce:             1,
		ExtraData:         make([]byte, 0),
	}

	tx := &types.Transaction{
		Data: types.TransactionData{
			From:    *crypto.MustGenerateRandomAddress(),
			To:      *crypto.MustGenerateRandomAddress(),
			Amount:  big.NewInt(3),
			Fee:     big.NewInt(0),
			Payload: make([]byte, 0),
		},
		Signature: crypto.Signature{Sig: []byte("test sig")},
	}

	tx.Hash = crypto.MustHash(tx.Data)

	block := &types.Block{
		HeaderHash:   header.Hash(),
		Header:       header,
		Transactions: []*types.Transaction{tx},
	}
	return block
}

func Test_GetBanlance(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".GetBanlance")

	api := newTestAPI(t, dbPath)
	defer func() {
		api.s.Stop()
		os.RemoveAll(dbPath)
	}()

	statedb, err := api.s.chain.GetCurrentState()
	assert.Equal(t, err, nil)

	// set coinbase banlance
	coinbase := api.s.miner.GetCoinbase()
	statedb.CreateAccount(coinbase)
	statedb.SetBalance(coinbase, common.SeeleToFan)

	// set randomAcct banlance
	randomAcct := getFromAddress(statedb)
	balance := big.NewInt(0).Sub(common.SeeleToFan, big.NewInt(2))
	statedb.SetBalance(randomAcct, balance)

	// save the statedb
	batch := api.s.accountStateDB.NewBatch()
	block := api.s.chain.CurrentBlock()
	block.Header.StateHash, _ = statedb.Commit(batch)
	block.Header.Height++
	block.Header.PreviousBlockHash = block.HeaderHash
	block.HeaderHash = block.Header.Hash()
	api.s.chain.GetStore().PutBlock(block, big.NewInt(1), true)
	batch.Commit()

	// get EmptyAddress balance
	resp, err := api.GetBalance(common.EmptyAddress)
	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Account, coinbase)
	assert.Equal(t, resp.Balance, common.SeeleToFan)

	// get randomAcct balance
	resp, err = api.GetBalance(randomAcct)
	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Account, randomAcct)
	assert.Equal(t, resp.Balance, balance)

	// get non-existence account balance
	randomAcct1 := getFromAddress(statedb)
	resp, err = api.GetBalance(randomAcct1)
	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Account, randomAcct1)
	assert.Equal(t, resp.Balance, big.NewInt(0))
}
