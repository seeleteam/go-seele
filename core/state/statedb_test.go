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

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/trie"
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
		statedb.GetOrNewStateObject(BytesToAddressForTest([]byte{i}))
		statedb.AddBalance(BytesToAddressForTest([]byte{i}), big.NewInt(4*int64(i)))
		statedb.SetNonce(BytesToAddressForTest([]byte{i}), 1)
	}

	hash, statedb := commitAndNewStateDB(statedb)

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
		stateobject := statedb.GetOrNewStateObject(BytesToAddressForTest([]byte{i}))
		nonce := stateobject.GetNonce()
		statedb.SubBalance(BytesToAddressForTest([]byte{i}), big.NewInt(2*int64(i)))
		stateobject.SetNonce(nonce + 1)
	}

	hash, statedb := commitAndNewStateDB(statedb)

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
		statedb.GetOrNewStateObject(BytesToAddressForTest([]byte{i}))
		nonce := statedb.GetNonce(BytesToAddressForTest([]byte{i}))
		statedb.SetBalance(BytesToAddressForTest([]byte{i}), big.NewInt(4*int64(i)))
		statedb.SetNonce(BytesToAddressForTest([]byte{i}), nonce+1)
	}

	hash, statedb := commitAndNewStateDB(statedb)

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
	statedb.SetState(addr, common.StringToHash("test key"), common.StringToHash("test value"))

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
	trie, err := trie.NewTrie(common.BytesToHash(stateObj.account.StorageRootHash), dbPrefixStorage, db)
	assert.Equal(t, err, nil)
	storageKey := stateObj.getStorageKey(common.StringToHash("test key"))
	storageValue, found := trie.Get(storageKey)
	assert.Equal(t, found, true)
	assert.Equal(t, storageValue, common.StringToHash("test value").Bytes())
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
		statedb.SetState(addr, common.StringToHash("key"), common.StringToHash("value"))

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

		if value := statedb2.GetState(addr, common.StringToHash("key")); !value.Equal(common.StringToHash("value")) {
			t.Fatalf("Invalid acocunt state value, %v", value.ToHex())
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
		statedb.SetState(addr, common.StringToHash("key"), common.StringToHash("value"))

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
