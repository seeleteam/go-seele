/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
)

func newTestStateDB() (database.Database, func()) {
	dir, err := ioutil.TempDir("", "teststatedb")
	if err != nil {
		panic(err)
	}
	db, err := leveldb.NewLevelDB(dir)
	if err != nil {
		panic(err)
	}
	return db, func() {
		db.Close()
		os.RemoveAll(dir)
	}
}

func BytesToAddressForTest(b []byte) common.Address {
	return common.BytesToAddress(b)
}

func Test_Statedb_Operate(t *testing.T) {
	db, remove := newTestStateDB()
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

func getAddr(a int) common.Address {
	return common.BytesToAddress([]byte(strconv.Itoa(a)))
}

func TestStatedb_Cache(t *testing.T) {
	db, remove := newTestStateDB()
	defer remove()
	statedb, err := NewStatedb(common.Hash{}, db)
	if err != nil {
		panic(err)
	}

	i := 0
	for ; i < StateCacheCapacity; i++ {
		state := statedb.GetOrNewStateObject(getAddr(i))

		if i == 0 {
			state.SetAmount(big.NewInt(4))
		}
	}

	assert.Equal(t, statedb.stateObjects.Len(), StateCacheCapacity)
	assert.Equal(t, statedb.trie.Hash(), common.Hash{})

	statedb.GetOrNewStateObject(getAddr(i))
	empty := statedb.getStateObject(BytesToAddressForTest([]byte{byte(0)}))
	if empty != nil {
		t.Error("empty should be nil")
	}

	assert.Equal(t, statedb.stateObjects.Len(), StateCacheCapacity*3/4+1)
	if statedb.trie.Hash() == common.EmptyHash {
		t.Error("trie root hash should changed")
	}
}
