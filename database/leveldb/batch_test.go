/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package leveldb

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/database"
	"github.com/syndtr/goleveldb/leveldb"
)

func prepareLevelDbFolder(pathRoot string, subDir string) string {
	dir, err := ioutil.TempDir(pathRoot, subDir)
	if err != nil {
		panic(err)
	}

	return dir
}

func newLevelDbInstance(dbPath string) database.Database {
	db, err := NewLevelDB(dbPath)
	if err != nil {
		panic(err)
	}

	return db
}

func newTestLevelDBForBatch() (database.Database, func()) {
	dir := prepareLevelDbFolder("", "leveldbtest")
	db := newLevelDbInstance(dir)
	return db, func() {
		db.Close()
		os.RemoveAll(dir)
	}
}

func Test_Batch_Put(t *testing.T) {
	// Init levelDB
	db, remove := newTestLevelDBForBatch()
	defer remove()

	batch := db.NewBatch()

	batch.Put([]byte("1"), []byte("11"))
	batch.Put([]byte("2"), []byte("22"))
	batch.Put([]byte("3"), []byte("33"))

	err := batch.Commit()
	if err != nil {
		t.Fatalf("Failed to commit batch")
	}
}

func Test_Batch_Delete(t *testing.T) {
	// Init levelDB
	db, remove := newTestLevelDBForBatch()
	defer remove()

	batch := db.NewBatch()

	batch.Put([]byte("1"), []byte("11"))
	batch.Put([]byte("2"), []byte("22"))
	batch.Put([]byte("3"), []byte("33"))
	batch.Delete([]byte("2"))
	err := batch.Commit()
	if err != nil {
		t.Fatalf("Failed to commit batch")
	}
	value, err := db.GetString("2")
	assert.Equal(t, err, leveldb.ErrNotFound)

	value, err = db.GetString("3")
	assert.Equal(t, value, "33")
}

func Test_Batch_Commit(t *testing.T) {
	// Init levelDB
	db, remove := newTestLevelDBForBatch()
	defer remove()

	batch := db.NewBatch()

	batch.Put([]byte("1"), []byte("11"))
	batch.Put([]byte("2"), []byte("22"))
	batch.Put([]byte("3"), []byte("33"))
	batch.Delete([]byte("2"))
	batch.Delete([]byte("1"))
	batch.Put([]byte("1"), []byte("1111"))
	value, err := db.GetString("1")
	assert.Equal(t, err, leveldb.ErrNotFound)
	batch.Put([]byte("1"), []byte("0000"))
	err = batch.Commit()
	if err != nil {
		t.Fatalf("Failed to commit batch")
	}

	value, err = db.GetString("1")
	assert.Equal(t, value, "0000")
	value, err = db.GetString("3")
	assert.Equal(t, value, "33")

	batch.Put([]byte("3"), []byte("3333"))
	batch.Put([]byte("1"), []byte("1111"))
	batch.Put([]byte("2"), []byte("2222"))

	err = batch.Commit()
	if err != nil {
		t.Fatalf("Failed to commit batch")
	}

	value, err = db.GetString("2")
	assert.Equal(t, value, "2222")
	value, err = db.GetString("3")
	assert.Equal(t, value, "3333")

}

func Test_Batch_Rollback(t *testing.T) {
	// Init levelDB
	db, remove := newTestLevelDBForBatch()
	defer remove()

	batch := db.NewBatch()

	batch.Put([]byte("1"), []byte("11"))
	batch.Put([]byte("2"), []byte("22"))
	batch.Put([]byte("3"), []byte("33"))
	err := batch.Commit()
	if err != nil {
		t.Fatalf("Failed to commit batch")
	}

	batch.Put([]byte("1"), []byte("1111"))
	batch.Put([]byte("2"), []byte("2222"))
	batch.Put([]byte("3"), []byte("3333"))

	batch.Rollback()

	value, err := db.GetString("2")
	assert.Equal(t, value, "22")
}
