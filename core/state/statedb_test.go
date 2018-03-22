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
	"testing"

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
	var a common.Address
	copy(a[:], b)
	return a
}

func Test_Statedb_Operate(t *testing.T) {
	db, remove := newTestStateDB()
	defer remove()

	hash := teststatedbaddmount(common.Hash{}, db)

	hash2 := teststatedbsubmount(hash, db)

	hash = teststatedbsetmount(hash2, db)

	statedb, err := NewStatedb(hash2, db) // for test old block
	if err != nil {
		panic(err)
	}
	for i := byte(0); i < 255; i++ {
		amount, _ := statedb.GetAmount(BytesToAddressForTest([]byte{i}))
		nonce, _ := statedb.GetNonce(BytesToAddressForTest([]byte{i}))
		if amount.Cmp(big.NewInt(2*int64(i))) != 0 {
			panic(fmt.Errorf("error anount amount %d", i))
		}
		if nonce != 2 {
			panic(fmt.Errorf("error anount nonce %d", i))
		}
	}

}

func teststatedbaddmount(root common.Hash, db database.Database) common.Hash {
	statedb, err := NewStatedb(common.Hash{}, db)
	if err != nil {
		panic(err)
	}
	for i := byte(0); i < 255; i++ {
		statedb.GetOrNewStateObject(BytesToAddressForTest([]byte{i}))
		statedb.AddAmount(BytesToAddressForTest([]byte{i}), big.NewInt(4*int64(i)))
		statedb.SetNonce(BytesToAddressForTest([]byte{i}), 1)
	}

	batch := db.NewBatch()
	hash, err := statedb.Commit(batch)
	if err != nil {
		panic(err)
	}
	batch.Commit()

	statedb, err = NewStatedb(hash, db)
	if err != nil {
		panic(err)
	}
	for i := byte(0); i < 255; i++ {
		amount, _ := statedb.GetAmount(BytesToAddressForTest([]byte{i}))
		nonce, _ := statedb.GetNonce(BytesToAddressForTest([]byte{i}))
		if amount.Cmp(big.NewInt(4*int64(i))) != 0 {
			panic(fmt.Errorf("error anount amount %d", i))
		}
		if nonce != 1 {
			panic(fmt.Errorf("error anount nonce %d", i))
		}
	}
	return hash
}

func teststatedbsubmount(root common.Hash, db database.Database) common.Hash {
	statedb, err := NewStatedb(root, db)
	if err != nil {
		panic(err)
	}
	for i := byte(0); i < 255; i++ {
		stateobject := statedb.GetOrNewStateObject(BytesToAddressForTest([]byte{i}))
		nonce := stateobject.GetNonce()
		statedb.SubAmount(BytesToAddressForTest([]byte{i}), big.NewInt(2*int64(i)))
		stateobject.SetNonce(nonce + 1)
	}

	batch := db.NewBatch()
	hash, err := statedb.Commit(batch)
	if err != nil {
		panic(err)
	}
	batch.Commit()

	statedb, err = NewStatedb(hash, db)
	if err != nil {
		panic(err)
	}
	for i := byte(0); i < 255; i++ {
		amount, _ := statedb.GetAmount(BytesToAddressForTest([]byte{i}))
		nonce, _ := statedb.GetNonce(BytesToAddressForTest([]byte{i}))
		if amount.Cmp(big.NewInt(2*int64(i))) != 0 {
			panic(fmt.Errorf("error anount amount %d", i))
		}
		if nonce != 2 {
			panic(fmt.Errorf("error anount nonce %d", i))
		}
	}
	return hash
}

func teststatedbsetmount(root common.Hash, db database.Database) common.Hash {
	statedb, err := NewStatedb(root, db)
	if err != nil {
		panic(err)
	}
	for i := byte(0); i < 255; i++ {
		statedb.GetOrNewStateObject(BytesToAddressForTest([]byte{i}))
		nonce, _ := statedb.GetNonce(BytesToAddressForTest([]byte{i}))
		statedb.SetAmount(BytesToAddressForTest([]byte{i}), big.NewInt(4*int64(i)))
		statedb.SetNonce(BytesToAddressForTest([]byte{i}), nonce+1)
	}

	batch := db.NewBatch()
	hash, err := statedb.Commit(batch)
	if err != nil {
		panic(err)
	}
	batch.Commit()

	statedb, err = NewStatedb(hash, db)
	if err != nil {
		panic(err)
	}
	for i := byte(0); i < 255; i++ {
		amount, _ := statedb.GetAmount(BytesToAddressForTest([]byte{i}))
		nonce, _ := statedb.GetNonce(BytesToAddressForTest([]byte{i}))
		if amount.Cmp(big.NewInt(4*int64(i))) != 0 {
			panic(fmt.Errorf("error anount amount %d", i))
		}
		if nonce != 3 {
			panic(fmt.Errorf("error anount nonce %d", i))
		}

		statedb.SetAmount(BytesToAddressForTest([]byte{i}), big.NewInt(4*int64(i)))
		statedb.SetNonce(BytesToAddressForTest([]byte{i}), nonce+1)
	}
	return hash
}
