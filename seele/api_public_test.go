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

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/node"
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
	log := log.GetLogger("seele", true)
	ss, err := NewSeeleService(ctx, conf, log)
	if err != nil {
		t.Fatal()
	}

	api := NewPublicSeeleAPI(ss)
	var info MinerInfo
	api.GetInfo(nil, &info)

	if !bytes.Equal(conf.SeeleConfig.Coinbase[0:], info.Coinbase[0:]) {
		t.Fail()
	}
}

func Test_Call(t *testing.T) {
	api := newTestAPI(t, filepath.Join(common.GetTempFolder(), ".Call"))
	// Create a simple_storage contract, get = 23
	contractAddress, from := createContract(t, api, "0x6080604052601760005534801561001557600080fd5b5060df806100246000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a72305820a184cfae11a459efce14d114b09674a03d70eec7e0e19586a38528392a74d4b20029")

	// The origin statedb
	statedbOri, err := api.s.chain.GetCurrentState()
	assert.Equal(t, err, nil)

	// Call the get function
	msg, err := hexutil.HexToBytes("0x6d4ce63c")
	assert.Equal(t, err, nil)
	getTx, err := types.NewMessageTransaction(from, contractAddress, big.NewInt(0), big.NewInt(0), 1, msg)
	assert.Equal(t, err, nil)

	// Verify the result
	result := make(map[string]interface{})
	request := CallRequest{
		Tx:     getTx,
		Height: -1,
	}
	err = api.Call(&request, &result)
	assert.Equal(t, err, nil)
	assert.Equal(t, result["result"], "0x0000000000000000000000000000000000000000000000000000000000000017")

	// It is no diffrence to the origin statedb
	statedbCur, err := api.s.chain.GetCurrentState()
	assert.Equal(t, err, nil)
	assert.Equal(t, statedbOri, statedbCur)
}

func newTestAPI(t *testing.T, dbPath string) *PublicSeeleAPI {
	conf := getTmpConfig()
	serviceContext := ServiceContext{
		DataDir: dbPath,
	}

	var key interface{} = "ServiceContext"
	ctx := context.WithValue(context.Background(), key, serviceContext)
	dataDir := ctx.Value("ServiceContext").(ServiceContext).DataDir
	defer os.RemoveAll(dataDir)

	log := log.GetLogger("seele", true)
	ss, err := NewSeeleService(ctx, conf, log)
	assert.Equal(t, err, nil)

	return NewPublicSeeleAPI(ss)
}

func createContract(t *testing.T, api *PublicSeeleAPI, payload string) (contractAddress common.Address, from common.Address) {
	// Construct a create contract transaction
	bytecode, err := hexutil.HexToBytes(payload)
	assert.Equal(t, err, nil)

	statedb, err := api.s.chain.GetCurrentState()
	assert.Equal(t, err, nil)

	from = getFromAddress(statedb)
	createContractTx, err := types.NewContractTransaction(from, big.NewInt(0), big.NewInt(0), 0, bytecode)
	assert.Equal(t, err, nil)

	receipt, err := api.s.chain.ApplyTransaction(createContractTx, 0, api.s.miner.GetCoinbase(), statedb, api.s.chain.CurrentBlock().Header)
	assert.Equal(t, err, nil)

	// Get the contract address
	contractAddressHex := hexutil.BytesToHex(receipt.ContractAddress)
	contractAddress, err = common.HexToAddress(contractAddressHex)
	assert.Equal(t, err, nil)

	// Save the statedb
	batch := api.s.accountStateDB.NewBatch()
	block := api.s.chain.CurrentBlock()
	block.Header.StateHash, _ = statedb.Commit(batch)
	batch.Commit()
	return contractAddress, from
}

func getFromAddress(statedb *state.Statedb) common.Address {
	from := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(from)
	statedb.SetBalance(from, common.SeeleToFan)
	statedb.SetNonce(from, 0)
	return from
}
