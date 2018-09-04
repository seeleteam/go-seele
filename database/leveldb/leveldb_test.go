/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package leveldb

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/seeleteam/go-seele/database"
	"github.com/syndtr/goleveldb/leveldb"
)

func Test_Put(t *testing.T) {
	// Init LevelDB
	dir := prepareDbFolder("", "leveldbtest")
	defer os.RemoveAll(dir)
	db := newDbInstance(dir)
	defer db.Close()

	// check insert and get
	err := db.PutString("1", "2")
	assert.Equal(t, err, nil)

	value, err := db.GetString("1")
	assert.Equal(t, err, nil)
	assert.Equal(t, value, "2")

	// Put empty key
	err = db.PutString("", "2")
	assert.Equal(t, err, ErrEmptyKey)
}

func Test_Has(t *testing.T) {
	// Init LevelDB
	dir := prepareDbFolder("", "leveldbtest")
	defer os.RemoveAll(dir)
	db := newDbInstance(dir)
	defer db.Close()

	// check whether key exists
	db.PutString("1", "2")
	exist, err := db.HasString("1")
	assert.Equal(t, err, nil)
	assert.Equal(t, exist, true)
}

func Test_Update(t *testing.T) {
	// Init LevelDB
	dir := prepareDbFolder("", "leveldbtest")
	defer os.RemoveAll(dir)
	db := newDbInstance(dir)
	defer db.Close()

	// check update and get
	db.PutString("1", "1")
	value, err := db.GetString("1")
	assert.Equal(t, err, nil)
	assert.Equal(t, value, "1")

	db.PutString("1", "3")
	value, err = db.GetString("1")
	assert.Equal(t, err, nil)
	assert.Equal(t, value, "3")
}

func Test_Delete(t *testing.T) {
	// Init LevelDB
	dir := prepareDbFolder("", "leveldbtest")
	defer os.RemoveAll(dir)
	db := newDbInstance(dir)
	defer db.Close()

	// insert and then delete key
	db.PutString("1", "1")
	err := db.DeleteSring("1")
	assert.Equal(t, err, nil)

	// check not found
	value, err := db.GetString("3")
	assert.Equal(t, err, leveldb.ErrNotFound)
	assert.Equal(t, value, "")

	// empty set
	exist, err := db.HasString("1")
	assert.Equal(t, err, nil)
	assert.Equal(t, exist, false)

	exist, err = db.HasString("3")
	assert.Equal(t, err, nil)
	assert.Equal(t, exist, false)
}

func Test_LevelDB_Newbatch(t *testing.T) {
	// Init levelDB
	dir := prepareDbFolder("", "leveldbtest")
	defer os.RemoveAll(dir)
	db := newDbInstance(dir)
	defer db.Close()

	batch := db.NewBatch()
	if batch == nil {
		t.Fatal("new level batch error")
	}
}

func prepareDbFolder(pathRoot string, subDir string) string {
	dir, err := ioutil.TempDir(pathRoot, subDir)
	if err != nil {
		panic(err)
	}

	return dir
}

func newDbInstance(dbPath string) database.Database {
	db, err := NewLevelDB(dbPath)
	if err != nil {
		panic(err)
	}

	return db
}
