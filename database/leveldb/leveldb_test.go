/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package leveldb

import (
	"github.com/magiconair/properties/assert"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"os"
	"testing"
)

func Test_LevelDB(t *testing.T) {
	dir, err := ioutil.TempDir("", "leveldbtest")
	if err != nil {
		panic(err)
	}

	defer os.RemoveAll(dir)

	db, err := NewLeveDB(dir)
	assert.Equal(t, err, nil)
	defer db.Close()

	err = db.PutString("1", "2")
	assert.Equal(t, err, nil)

	value, err := db.GetString("1")
	assert.Equal(t, err, nil)
	assert.Equal(t, value, "2")

	// check not found
	value, err = db.GetString("3")
	assert.Equal(t, err, leveldb.ErrNotFound)
	assert.Equal(t, value, "")
}
