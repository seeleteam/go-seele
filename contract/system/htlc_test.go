package system

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
)

var (
	secret       = "0xc5543fa77c58024c27879360b1fcd3fa67f546c3ebdc5f3598c30d10266e2837"
	forgedSecret = "0xc5543fa77c58024c27879360b1fcd3fa67f546c3ebdc5f3598c30d10266e2830"
	secretehash  = "0x20239be7188a95499bb9c96c848dd7815dce1819a74e12b0610d6e961c08e92b"
)

type testAccount struct {
	addr    common.Address
	privKey *ecdsa.PrivateKey
	amount  *big.Int
	nonce   uint64
}

var testGenesisAccounts = []*testAccount{
	newTestAccount(big.NewInt(100000), 0),
	newTestAccount(big.NewInt(100000), 0),
	newTestAccount(big.NewInt(100000), 0),
}

func newTestAccount(amount *big.Int, nonce uint64) *testAccount {
	addr, privKey, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(err)
	}

	return &testAccount{
		addr:    *addr,
		privKey: privKey,
		amount:  new(big.Int).Set(amount),
		nonce:   nonce,
	}
}

func newTestTx(from, to int, amount, fee, nonce uint64) *types.Transaction {
	fromAccount := testGenesisAccounts[from]
	toAccmount := testGenesisAccounts[to]
	tx, _ := types.NewTransaction(fromAccount.addr, toAccmount.addr, new(big.Int).SetUint64(amount), new(big.Int).SetUint64(fee), nonce)
	tx.Sign(fromAccount.privKey)

	return tx
}

func newContext(db database.Database, from, to int) *Context {
	tx := newTestTx(from, to, 100, 200, 1)
	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		panic(err)
	}

	return NewContext(tx, statedb)
}

func Test_NewContract(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	context := newContext(db, 0, 1)
	context.statedb.CreateAccount(testGenesisAccounts[0].addr)
	context.statedb.SetBalance(testGenesisAccounts[0].addr, big.NewInt(50000))
	var lockinfo lock
	locktime := time.Now().Unix() + 48*3600
	lockinfo.Timelock = uint64(locktime)
	lockinfo.Hashlock = secretehash
	databytes, err := json.Marshal(lockinfo)
	assert.Equal(t, err, nil)
	_, err = newContract(databytes, context)
	assert.Equal(t, err, nil)

	amount := context.statedb.GetBalance(testGenesisAccounts[0].addr)
	result := amount.Cmp(big.NewInt(50000 - 100))
	assert.Equal(t, result, 0)
}

func Test_Withdraw(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	context := newContext(db, 0, 1)
	context.statedb.CreateAccount(testGenesisAccounts[0].addr)
	context.statedb.SetBalance(testGenesisAccounts[0].addr, big.NewInt(50000))
	context.statedb.CreateAccount(testGenesisAccounts[1].addr)
	context.statedb.SetBalance(testGenesisAccounts[1].addr, big.NewInt(50000))

	amount := context.statedb.GetBalance(testGenesisAccounts[0].addr)
	result := amount.Cmp(big.NewInt(50000))
	assert.Equal(t, result, 0)

	amount = context.statedb.GetBalance(testGenesisAccounts[1].addr)
	result = amount.Cmp(big.NewInt(50000))
	assert.Equal(t, result, 0)

	var lockinfo lock
	locktime := time.Now().Unix() + 48*3600
	lockinfo.Timelock = uint64(locktime)
	lockinfo.Hashlock = secretehash
	databytes, err := json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	hashbytes, err := newContract(databytes, context)
	assert.Equal(t, err, nil)

	amount = context.statedb.GetBalance(testGenesisAccounts[0].addr)
	result = amount.Cmp(big.NewInt(50000 - 100))
	assert.Equal(t, result, 0)

	hash := common.BytesToHash(hashbytes)
	var withdrawInfo withdrawing
	withdrawInfo.Preimage = forgedSecret
	withdrawInfo.Hash = hash
	databytes, err = json.Marshal(withdrawInfo)
	assert.Equal(t, err, nil)

	// case 1: forged preimage
	_, err = withdraw(databytes, context)
	assert.Equal(t, err, fmt.Errorf("Failed to match the hashlock\n"))

	// case 2: forged receiver
	withdrawInfo.Preimage = secret
	databytes, err = json.Marshal(withdrawInfo)
	assert.Equal(t, err, nil)

	_, err = withdraw(databytes, context)
	assert.Equal(t, err, fmt.Errorf("Failed for you is not the real receiver\n"))

	// case 3: real receiver
	tx := newTestTx(1, 0, 100, 200, 0)
	context.tx = tx
	_, err = withdraw(databytes, context)
	assert.Equal(t, err, nil)

	amount = context.statedb.GetBalance(testGenesisAccounts[1].addr)
	result = amount.Cmp(big.NewInt(50000 + 100))
	assert.Equal(t, result, 0)

	// case 4: already withrawed
	_, err = withdraw(databytes, context)
	assert.Equal(t, err, fmt.Errorf("Failed for already withdrawed\n"))

	// case 5: timelock is passed, can not be withdrawable
	locktime = time.Now().Unix() + 1
	lockinfo.Timelock = uint64(locktime)
	databytes, err = json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	hashbytes, err = newContract(databytes, context)
	assert.Equal(t, err, nil)
	amount = context.statedb.GetBalance(testGenesisAccounts[1].addr)
	result = amount.Cmp(big.NewInt(50000))
	assert.Equal(t, result, 0)

	hash = common.BytesToHash(hashbytes)
	withdrawInfo.Preimage = secret
	withdrawInfo.Hash = hash
	databytes, err = json.Marshal(withdrawInfo)
	assert.Equal(t, err, nil)

	tx = newTestTx(0, 1, 100, 100, 0)
	context.tx = tx
	time.Sleep(1 * time.Second)
	_, err = withdraw(databytes, context)
	assert.Equal(t, err, fmt.Errorf("Failed for timelock is passed\n"))
}

func Test_Refund(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	context := newContext(db, 0, 1)
	context.statedb.CreateAccount(testGenesisAccounts[0].addr)
	context.statedb.SetBalance(testGenesisAccounts[0].addr, big.NewInt(50000))
	context.statedb.CreateAccount(testGenesisAccounts[1].addr)
	context.statedb.SetBalance(testGenesisAccounts[1].addr, big.NewInt(50000))

	amount := context.statedb.GetBalance(testGenesisAccounts[0].addr)
	result := amount.Cmp(big.NewInt(50000))
	assert.Equal(t, result, 0)

	amount = context.statedb.GetBalance(testGenesisAccounts[1].addr)
	result = amount.Cmp(big.NewInt(50000))
	assert.Equal(t, result, 0)

	var lockinfo lock
	locktime := time.Now().Unix() + 48*3600
	lockinfo.Timelock = uint64(locktime)
	lockinfo.Hashlock = secretehash
	databytes, err := json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	hashbytes, err := newContract(databytes, context)
	assert.Equal(t, err, nil)

	amount = context.statedb.GetBalance(testGenesisAccounts[0].addr)
	result = amount.Cmp(big.NewInt(50000 - 100))
	assert.Equal(t, result, 0)

	// case 1: forged sender
	tx := newTestTx(1, 0, 100, 100, 0)
	context.tx = tx
	_, err = refund(hashbytes, context)
	assert.Equal(t, err, fmt.Errorf("Failed for you is not the sender\n"))

	// case 2: timelock is not over
	tx = newTestTx(0, 1, 100, 100, 0)
	context.tx = tx
	_, err = refund(hashbytes, context)
	assert.Equal(t, err, fmt.Errorf("Failed for timelock is not over\n"))

	// case 3: receiver have withdrawed
	locktime = time.Now().Unix() + 1
	lockinfo.Timelock = uint64(locktime)
	databytes, err = json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	hashbytes, err = newContract(databytes, context)
	assert.Equal(t, err, nil)

	amount = context.statedb.GetBalance(testGenesisAccounts[0].addr)
	result = amount.Cmp(big.NewInt(50000 - 100 - 100))
	assert.Equal(t, result, 0)

	tx = newTestTx(1, 0, 100, 100, 0)
	context.tx = tx
	hash := common.BytesToHash(hashbytes)
	var withdrawInfo withdrawing
	withdrawInfo.Preimage = secret
	withdrawInfo.Hash = hash
	databytes, err = json.Marshal(withdrawInfo)
	assert.Equal(t, err, nil)
	_, err = withdraw(databytes, context)
	assert.Equal(t, err, nil)

	tx = newTestTx(0, 1, 100, 1, 0)
	context.tx = tx
	time.Sleep(1 * time.Second)
	_, err = refund(hashbytes, context)
	assert.Equal(t, err, fmt.Errorf("Failed for receiver have withdrawed\n"))

	// case 4: refund
	locktime = time.Now().Unix() + 1
	lockinfo.Timelock = uint64(locktime)
	databytes, err = json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	hashbytes, err = newContract(databytes, context)
	assert.Equal(t, err, nil)

	amount = context.statedb.GetBalance(testGenesisAccounts[0].addr)
	result = amount.Cmp(big.NewInt(50000 - 100 - 100 - 100))
	assert.Equal(t, result, 0)

	time.Sleep(1 * time.Second)
	_, err = refund(hashbytes, context)
	assert.Equal(t, err, nil)

	// case 5: already been refunded
	_, err = refund(hashbytes, context)
	assert.Equal(t, err, fmt.Errorf("Failed for receiver have refunded\n"))
}

func Test_GetContract(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	context := newContext(db, 0, 1)
	context.statedb.CreateAccount(testGenesisAccounts[0].addr)
	context.statedb.SetBalance(testGenesisAccounts[0].addr, big.NewInt(50000))
	context.statedb.CreateAccount(hashTimeLockContractAddress)
	var lockinfo lock
	locktime := time.Now().Unix() + 48*3600
	lockinfo.Timelock = uint64(locktime)
	lockinfo.Hashlock = secretehash
	databytes, err := json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	// case 1: get data by key
	hashbytes, err := newContract(databytes, context)
	_, err = getContract(hashbytes, context)
	assert.Equal(t, err, nil)

	// case 2: get data by key, no value with key
	_, err = getContract(common.EmptyHash.Bytes(), context)
	assert.Equal(t, err, fmt.Errorf("Faild for no value with the key\n"))

}
