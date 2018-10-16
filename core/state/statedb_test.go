/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/trie"
	"github.com/stretchr/testify/assert"
)

func BytesToAddressForTest(b []byte) common.Address {
	return common.BytesToAddress(b)
}

func Test_Statedb_Operate(t *testing.T) {
	db, remove := leveldb.NewTestDatabase()
	defer remove()

	hash := teststatedbaddbalance(common.Hash{}, db)

	hash2 := teststatedbsubbalance(hash, db)

	hash = teststatedbsetbalance(hash2, db)

	statedb, err := NewStatedb(hash2, db) // for test old block
	if err != nil {
		panic(err)
	}
	for i := byte(0); i < 255; i++ {
		balance := statedb.GetBalance(BytesToAddressForTest([]byte{i}))
		nonce := statedb.GetNonce(BytesToAddressForTest([]byte{i}))
		if balance.Cmp(big.NewInt(2*int64(i))) != 0 {
			panic(fmt.Errorf("error anount balance %d", i))
		}
		if nonce != 2 {
			panic(fmt.Errorf("error anount nonce %d", i))
		}
	}

}

func teststatedbaddbalance(root common.Hash, db database.Database) common.Hash {
	statedb, err := NewStatedb(common.Hash{}, db)
	if err != nil {
		panic(err)
	}
	for i := byte(0); i < 255; i++ {
		statedb.CreateAccount(BytesToAddressForTest([]byte{i}))
		statedb.AddBalance(BytesToAddressForTest([]byte{i}), big.NewInt(4*int64(i)))
		statedb.SetNonce(BytesToAddressForTest([]byte{i}), 1)
	}

	hash, statedb := commitAndNewStateDB(db, statedb)

	for i := byte(0); i < 255; i++ {
		balance := statedb.GetBalance(BytesToAddressForTest([]byte{i}))
		nonce := statedb.GetNonce(BytesToAddressForTest([]byte{i}))
		if balance.Cmp(big.NewInt(4*int64(i))) != 0 {
			panic(fmt.Errorf("error anount balance %d", i))
		}
		if nonce != 1 {
			panic(fmt.Errorf("error anount nonce %d", i))
		}
	}
	return hash
}

func teststatedbsubbalance(root common.Hash, db database.Database) common.Hash {
	statedb, err := NewStatedb(root, db)
	if err != nil {
		panic(err)
	}
	for i := byte(0); i < 255; i++ {
		statedb.CreateAccount(BytesToAddressForTest([]byte{i}))
		nonce := statedb.GetNonce(BytesToAddressForTest([]byte{i}))
		statedb.SubBalance(BytesToAddressForTest([]byte{i}), big.NewInt(2*int64(i)))
		statedb.SetNonce(BytesToAddressForTest([]byte{i}), nonce+1)
	}

	hash, statedb := commitAndNewStateDB(db, statedb)

	for i := byte(0); i < 255; i++ {
		balance := statedb.GetBalance(BytesToAddressForTest([]byte{i}))
		nonce := statedb.GetNonce(BytesToAddressForTest([]byte{i}))
		if balance.Cmp(big.NewInt(2*int64(i))) != 0 {
			panic(fmt.Errorf("error anount balance %d", i))
		}
		if nonce != 2 {
			panic(fmt.Errorf("error anount nonce %d", i))
		}
	}
	return hash
}

func teststatedbsetbalance(root common.Hash, db database.Database) common.Hash {
	statedb, err := NewStatedb(root, db)
	if err != nil {
		panic(err)
	}
	for i := byte(0); i < 255; i++ {
		statedb.CreateAccount(BytesToAddressForTest([]byte{i}))
		nonce := statedb.GetNonce(BytesToAddressForTest([]byte{i}))
		statedb.SetBalance(BytesToAddressForTest([]byte{i}), big.NewInt(4*int64(i)))
		statedb.SetNonce(BytesToAddressForTest([]byte{i}), nonce+1)
	}

	hash, statedb := commitAndNewStateDB(db, statedb)

	for i := byte(0); i < 255; i++ {
		balance := statedb.GetBalance(BytesToAddressForTest([]byte{i}))
		nonce := statedb.GetNonce(BytesToAddressForTest([]byte{i}))
		if balance.Cmp(big.NewInt(4*int64(i))) != 0 {
			panic(fmt.Errorf("error anount balance %d", i))
		}
		if nonce != 3 {
			panic(fmt.Errorf("error anount nonce %d", i))
		}

		statedb.SetBalance(BytesToAddressForTest([]byte{i}), big.NewInt(4*int64(i)))
		statedb.SetNonce(BytesToAddressForTest([]byte{i}), nonce+1)
	}
	return hash
}

func Test_Commit_AccountStorages(t *testing.T) {
	db, remove := leveldb.NewTestDatabase()
	defer remove()

	statedb, err := NewStatedb(common.EmptyHash, db)
	assert.Equal(t, err, nil)

	addr := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(addr)
	statedb.SetBalance(addr, big.NewInt(99))
	statedb.SetNonce(addr, 38)
	statedb.SetCode(addr, []byte("test code"))
	statedb.SetData(addr, common.StringToHash("test key"), []byte("test value"))

	// Get root hash for receipt PostState
	root1, err := statedb.Hash()
	assert.Equal(t, err, nil)

	// Commit to DB
	batch := db.NewBatch()
	root2, err := statedb.Commit(batch)
	assert.Equal(t, err, nil)
	assert.Equal(t, root1, root2)
	assert.Equal(t, batch.Commit(), nil)

	stateObj := statedb.getStateObject(addr)

	// Validate state trie of created account
	trie, err := trie.NewTrie(root1, TrieDbPrefix, db)
	assert.Equal(t, err, nil)
	storageKey := stateObj.dataKey(dataTypeStorage, crypto.MustHash(common.StringToHash("test key")).Bytes()...)
	storageValue, found, err := trie.Get(storageKey)
	assert.Nil(t, err)
	assert.Equal(t, found, true)
	assert.Equal(t, storageValue, []byte("test value"))
}

func Test_StateDB_CommitMultipleChanges(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	statedb, err := NewStatedb(common.EmptyHash, db)
	assert.Equal(t, err, nil)

	var allAddr []common.Address

	// create multiple accounts with code and states
	for i := 0; i < 1500; i++ {
		addr := *crypto.MustGenerateRandomAddress()
		statedb.CreateAccount(addr)
		statedb.SetBalance(addr, big.NewInt(38))
		statedb.SetNonce(addr, 6)
		statedb.SetCode(addr, []byte("hello"))
		statedb.SetData(addr, common.StringToHash("key"), []byte("value"))

		if _, err = statedb.Hash(); err != nil {
			panic(err)
		}

		allAddr = append(allAddr, addr)
	}

	// serialize the new created accounts into DB
	batch := db.NewBatch()
	root, err := statedb.Commit(batch)
	assert.Equal(t, err, nil)
	assert.Equal(t, batch.Commit(), nil)

	// ensure all accounts could be loaded again with new statedb
	statedb2, err := NewStatedb(root, db)
	assert.Equal(t, err, nil)
	for i, addr := range allAddr {
		if !statedb2.Exist(addr) {
			t.Fatalf("Cannot find the inserted account, index = %v", i)
		}

		if balance := statedb2.GetBalance(addr).Int64(); balance != 38 {
			t.Fatalf("Invalid account balance %v", balance)
		}

		if nonce := statedb2.GetNonce(addr); nonce != 6 {
			t.Fatalf("Invalid account nonce %v", nonce)
		}

		if code := statedb2.GetCode(addr); !bytes.Equal(code, []byte("hello")) {
			t.Fatalf("Invalid account code %v", code)
		}

		if value := statedb2.GetData(addr, common.StringToHash("key")); string(value) != "value" {
			t.Fatalf("Invalid acocunt state value, %v", value)
		}
	}
}

func Benchmark_Trie_Hash(b *testing.B) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	statedb, err := NewStatedb(common.EmptyHash, db)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 5000; i++ {
		addr := common.BytesToAddress([]byte(strconv.Itoa(i)))

		statedb.CreateAccount(addr)
		statedb.SetBalance(addr, big.NewInt(38))
		statedb.SetNonce(addr, 6)
		statedb.SetCode(addr, []byte("hello"))
		statedb.SetData(addr, common.StringToHash("key"), []byte("value"))

		if _, err := statedb.Hash(); err != nil {
			panic(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := statedb.Hash(); err != nil {
			panic(err)
		}
	}
}

func newTestEVMStateDB() (database.Database, *Statedb, *stateObject, func()) {
	db, dispose := leveldb.NewTestDatabase()

	statedb, err := NewStatedb(common.EmptyHash, db)
	if err != nil {
		dispose()
		panic(err)
	}

	testAddr := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(testAddr)

	stateObj := statedb.getStateObject(testAddr)
	if stateObj == nil {
		dispose()
		panic("cannot find the state object.")
	}

	if !stateObj.address.Equal(testAddr) {
		dispose()
		panic("the address of state object is invalid.")
	}

	return db, statedb, stateObj, dispose
}

func commitAndNewStateDB(db database.Database, statedb *Statedb) (common.Hash, *Statedb) {
	batch := db.NewBatch()
	rootHash, err := statedb.Commit(batch)
	if err != nil {
		panic(err)
	}

	if err = batch.Commit(); err != nil {
		panic(err)
	}

	newStatedb, err := NewStatedb(rootHash, db)
	if err != nil {
		panic(err)
	}

	return rootHash, newStatedb
}

func Test_CreateAccount(t *testing.T) {
	_, statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	// Assert the no code for a new created account.
	assert.Equal(t, statedb.dbErr, error(nil))
	assert.Equal(t, stateObj.account.CodeHash, []byte(nil))
	assert.Equal(t, stateObj.code, []byte(nil))
	assert.Equal(t, stateObj.dirtyCode, false)
}

func Test_Code(t *testing.T) {
	db, statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	addr := stateObj.address

	// Set code to account
	code := []byte("test code")
	codeHash := crypto.HashBytes(code)
	statedb.SetCode(addr, code)

	// Validate code APIs
	assert.Equal(t, statedb.GetCodeHash(addr), codeHash)
	assert.Equal(t, statedb.GetCode(addr), code)
	assert.Equal(t, statedb.GetCodeSize(addr), len(code))
	assert.Equal(t, stateObj.dirtyCode, true)

	// Commit the account code change and create another state DB with the same root hash.
	_, statedb2 := commitAndNewStateDB(db, statedb)
	assert.Equal(t, stateObj.dirtyCode, false)

	// Ensure the account code is valid.
	assert.Equal(t, statedb2.GetCodeHash(addr), codeHash)
	assert.Equal(t, statedb2.GetCode(addr), code)
	assert.Equal(t, statedb2.GetCodeSize(addr), len(code))
	assert.Equal(t, stateObj.dirtyCode, false)

	// Empty address
	var emptyCode []byte
	addr = common.EmptyAddress
	assert.Equal(t, statedb2.GetCodeHash(addr), common.EmptyHash)
	assert.Equal(t, statedb2.GetCode(addr), emptyCode)
	assert.Equal(t, statedb2.GetCodeSize(addr), 0)
	assert.Equal(t, statedb2.Empty(addr), true)

	// An address that does not exist
	addr = *crypto.MustGenerateRandomAddress()
	assert.Equal(t, statedb2.GetCodeHash(addr), common.EmptyHash)
	assert.Equal(t, statedb2.GetCode(addr), emptyCode)
	assert.Equal(t, statedb2.GetCodeSize(addr), 0)
	assert.Equal(t, statedb2.Empty(addr), true)
}

func Test_Refund(t *testing.T) {
	_, statedb, _, dispose := newTestEVMStateDB()
	defer dispose()

	assert.Equal(t, statedb.refund, uint64(0))

	statedb.AddRefund(38)
	assert.Equal(t, statedb.refund, uint64(38))

	statedb.AddRefund(66)
	assert.Equal(t, statedb.refund, uint64(38+66))
}

func Test_Suicide(t *testing.T) {
	db, statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	addr := stateObj.address
	statedb.SetCode(addr, []byte("hello,world"))
	statedb.SetData(addr, common.StringToHash("k1"), []byte("v1"))
	statedb.SetData(addr, common.StringToHash("k2"), []byte("v2"))

	assert.Equal(t, statedb.Exist(addr), true)
	assert.Equal(t, statedb.HasSuicided(addr), false)

	assert.Equal(t, statedb.Suicide(*crypto.MustGenerateRandomAddress()), false)
	assert.Equal(t, statedb.Suicide(addr), true)
	assert.Equal(t, statedb.HasSuicided(addr), true)

	// Commit the state change
	_, statedb2 := commitAndNewStateDB(db, statedb)

	assert.Equal(t, statedb2.Exist(addr), false)                                    // account not exists
	assert.Equal(t, statedb2.GetCode(addr), []byte(nil))                            // code not exists
	assert.Equal(t, statedb2.GetData(addr, common.StringToHash("k1")), []byte(nil)) // k1 not exists
	assert.Equal(t, statedb2.GetData(addr, common.StringToHash("k2")), []byte(nil)) // k2 not exists
}

func Test_Log(t *testing.T) {
	_, statedb, _, dispose := newTestEVMStateDB()
	defer dispose()

	statedb.Prepare(38)

	statedb.AddLog(new(types.Log))
	statedb.AddLog(new(types.Log))
	statedb.AddLog(new(types.Log))

	logs := statedb.GetCurrentLogs()
	assert.Equal(t, len(logs), 3)
	assert.Equal(t, logs[0].TxIndex, uint(38))
	assert.Equal(t, logs[1].TxIndex, uint(38))
	assert.Equal(t, logs[2].TxIndex, uint(38))
}
