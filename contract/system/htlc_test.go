/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package system

import (
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

var (
	secret       = "0x31aa0be185fbc89048a0381cc5136969565be9d9c13048f7c2ee9322811b99fc"
	forgedSecret = "0xc5543fa77c58024c27879360b1fcd3fa67f546c3ebdc5f3598c30d10266e2830"
	secretehash  = "0x57e685963f607851af252e7922483a61fbceced12accd745444f412295517768"
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

func newTestTx(from, to int, amount, price, nonce uint64) *types.Transaction {
	fromAccount := testGenesisAccounts[from]
	toAccmount := testGenesisAccounts[to]
	tx, _ := types.NewTransaction(fromAccount.addr, toAccmount.addr, new(big.Int).SetUint64(amount), new(big.Int).SetUint64(price), nonce)
	tx.Sign(fromAccount.privKey)

	return tx
}

func newContext(db database.Database, from, to int) *Context {
	tx := newTestTx(from, to, 100, 200, 1)
	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		panic(err)
	}

	return NewContext(tx, statedb, newTestBlockHeader())
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
		Witness:           make([]byte, 0),
		ExtraData:         make([]byte, 0),
	}

}

func Test_newHTLC(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	context := newContext(db, 0, 1)
	context.statedb.CreateAccount(testGenesisAccounts[0].addr)
	context.statedb.SetBalance(testGenesisAccounts[0].addr, big.NewInt(50000))
	var lockinfo HashTimeLock
	locktime := time.Now().Unix() + 48*3600
	lockinfo.TimeLock = locktime
	hash, err := hexutil.HexToBytes(secretehash)
	assert.Equal(t, err, nil)

	lockinfo.HashLock = hash
	databytes, err := json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	_, err = newHTLC(databytes, context)
	assert.Equal(t, err, nil)

	amount := context.statedb.GetBalance(testGenesisAccounts[0].addr)
	result := amount.Cmp(big.NewInt(50000))
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

	var lockinfo HashTimeLock
	locktime := time.Now().Unix() + 48*3600
	lockinfo.TimeLock = locktime
	hash, err := hexutil.HexToBytes(secretehash)
	assert.Equal(t, err, nil)

	lockinfo.HashLock = hash
	lockinfo.To = context.tx.Data.To
	databytes, err := json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	_, err = newHTLC(databytes, context)
	assert.Equal(t, err, nil)

	key := context.tx.Hash
	var withdrawInfo Withdrawing
	preimage, err := hexutil.HexToBytes(forgedSecret)
	assert.Equal(t, err, nil)

	withdrawInfo.Preimage = preimage
	withdrawInfo.Hash = key
	databytes, err = json.Marshal(withdrawInfo)
	assert.Equal(t, err, nil)

	// case 1: forged preimage
	_, err = withdraw(databytes, context)
	assert.Equal(t, err, errHashMismatch)

	// case 2: forged receiver
	preimage, err = hexutil.HexToBytes(secret)
	assert.Equal(t, err, nil)

	withdrawInfo.Preimage = preimage
	databytes, err = json.Marshal(withdrawInfo)
	assert.Equal(t, err, nil)

	_, err = withdraw(databytes, context)
	assert.Equal(t, err, errReceiver)

	// case 3: real receiver
	tx := newTestTx(1, 0, 100, 200, 0)
	context.tx = tx
	_, err = withdraw(databytes, context)
	assert.Equal(t, err, nil)

	amount = context.statedb.GetBalance(testGenesisAccounts[1].addr)
	result = amount.Cmp(big.NewInt(50000 + 100))
	assert.Equal(t, result, 0)

	amount = context.statedb.GetBalance(testGenesisAccounts[0].addr)
	result = amount.Cmp(big.NewInt(50000 - 100))
	assert.Equal(t, result, 0)

	// case 4: already withrawed
	_, err = withdraw(databytes, context)
	assert.Equal(t, err, errWithdrawAfterWithdrawed)

	// case 5: timelock is passed, can not be withdrawed
	locktime = time.Now().Unix() + 1
	lockinfo.TimeLock = locktime
	lockinfo.To = context.tx.Data.To
	databytes, err = json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	_, err = newHTLC(databytes, context)
	assert.Equal(t, err, nil)

	key = context.tx.Hash
	preimage, err = hexutil.HexToBytes(secret)
	assert.Equal(t, err, nil)

	withdrawInfo.Preimage = preimage
	withdrawInfo.Hash = key
	databytes, err = json.Marshal(withdrawInfo)
	assert.Equal(t, err, nil)

	tx = newTestTx(0, 1, 100, 100, 0)
	context.tx = tx
	context.BlockHeader.CreateTimestamp = big.NewInt(time.Now().Unix() + 1)
	_, err = withdraw(databytes, context)
	assert.Equal(t, err, errTimeExpired)
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

	var lockinfo HashTimeLock
	locktime := time.Now().Unix() + 48*3600
	lockinfo.TimeLock = locktime
	hash, err := hexutil.HexToBytes(secretehash)
	assert.Equal(t, err, nil)
	lockinfo.HashLock = hash
	lockinfo.To = context.tx.Data.To
	databytes, err := json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	_, err = newHTLC(databytes, context)
	assert.Equal(t, err, nil)

	key := context.tx.Hash
	// case 1: forged sender
	tx := newTestTx(1, 0, 100, 100, 0)
	context.tx = tx
	_, err = refund(key.Bytes(), context)
	assert.Equal(t, err, errSender)

	// case 2: timelock is not over
	tx = newTestTx(0, 1, 100, 100, 0)
	context.tx = tx
	_, err = refund(key.Bytes(), context)
	assert.Equal(t, err, errTimeLocked)

	// case 3: receiver have withdrawed
	locktime = time.Now().Unix() + 1
	lockinfo.TimeLock = locktime
	databytes, err = json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	_, err = newHTLC(databytes, context)
	assert.Equal(t, err, nil)

	key = context.tx.Hash
	tx = newTestTx(1, 0, 100, 100, 0)
	context.tx = tx
	var withdrawInfo Withdrawing
	preimage, err := hexutil.HexToBytes(secret)
	assert.Equal(t, err, nil)

	withdrawInfo.Preimage = preimage
	withdrawInfo.Hash = key
	databytes, err = json.Marshal(withdrawInfo)
	assert.Equal(t, err, nil)

	_, err = withdraw(databytes, context)
	assert.Equal(t, err, nil)

	tx = newTestTx(0, 1, 100, 1, 0)
	context.tx = tx
	context.BlockHeader.CreateTimestamp = big.NewInt(locktime + 2)
	_, err = refund(key.Bytes(), context)
	assert.Equal(t, err, errRefundAfterWithdrawed)

	// case 4: refund
	context.BlockHeader.CreateTimestamp = big.NewInt(time.Now().Unix())
	locktime = time.Now().Unix() + 1
	lockinfo.TimeLock = locktime
	databytes, err = json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	_, err = newHTLC(databytes, context)
	assert.Equal(t, err, nil)

	key = context.tx.Hash
	context.BlockHeader.CreateTimestamp = big.NewInt(locktime + 2)
	_, err = refund(key.Bytes(), context)
	assert.Equal(t, err, nil)

	// case 5: already been refunded
	_, err = refund(key.Bytes(), context)
	assert.Equal(t, err, errRedunedAgain)
}

func Test_GetContract(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	context := newContext(db, 0, 1)
	context.statedb.CreateAccount(testGenesisAccounts[0].addr)
	context.statedb.SetBalance(testGenesisAccounts[0].addr, big.NewInt(50000))
	context.statedb.CreateAccount(HashTimeLockContractAddress)
	var lockinfo HashTimeLock
	locktime := time.Now().Unix() + 48*3600
	lockinfo.TimeLock = locktime
	hash, err := hexutil.HexToBytes(secretehash)
	assert.Equal(t, err, nil)

	lockinfo.HashLock = hash
	databytes, err := json.Marshal(lockinfo)
	assert.Equal(t, err, nil)

	// case 1: get data by key
	_, err = newHTLC(databytes, context)
	assert.Equal(t, err, nil)

	_, err = getContract(context.tx.Hash.Bytes(), context)
	assert.Equal(t, err, nil)

	// case 2: get data by key, no value with key
	_, err = getContract(common.EmptyHash.Bytes(), context)
	assert.Equal(t, err, errNotFound)
}
