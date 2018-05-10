/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
)

func newTestDatabase() (db database.Database, dispose func()) {
	dir, err := ioutil.TempDir("", "BlockchainCore")
	if err != nil {
		panic(err)
	}

	db, err = leveldb.NewLevelDB(dir)
	if err != nil {
		os.RemoveAll(dir)
		panic(err)
	}

	return db, func() {
		db.Close()
		os.RemoveAll(dir)
	}
}

func Test_Genesis_DefaultGenesis(t *testing.T) {
	default1 := DefaultGenesis(nil)
	default2 := DefaultGenesis(nil)

	assert.Equal(t, default1.header, default2.header)
}

func Test_Genesis_Init_DefaultGenesis(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bcStore := store.NewBlockchainDatabase(db)

	genesis := DefaultGenesis(bcStore)
	genesisHash := genesis.header.Hash()

	err := genesis.Initialize(db)
	if err != nil {
		panic(err)
	}

	hash, err := bcStore.GetBlockHash(genesisBlockHeight)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, hash, genesisHash)

	headHash, err := bcStore.GetHeadBlockHash()
	assert.Equal(t, err, error(nil))
	assert.Equal(t, headHash, genesisHash)

	_, err = state.NewStatedb(genesis.header.StateHash, db)
	assert.Equal(t, err, error(nil))
}

func Test_Genesis_Init_GenesisMismatch(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bcStore := store.NewBlockchainDatabase(db)

	header := DefaultGenesis(bcStore).header.Clone()
	header.Nonce = 38
	bcStore.PutBlockHeader(header.Hash(), header, header.Difficulty, true)

	genesis := DefaultGenesis(bcStore)
	err := genesis.Initialize(db)
	assert.Equal(t, err, ErrGenesisHashMismatch)
}

func Test_GenesisInfo(t *testing.T) {
	info := GenesisInfo{
		Difficulty: 100000,
		Accounts: map[string]uint64{
			"a": 1,
			"b": 2,
		},
	}

	result, _ := json.Marshal(info)
	fmt.Println(string(result))
}
