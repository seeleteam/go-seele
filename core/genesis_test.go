/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/database/leveldb"
)

func testBlockchainDatabase(ut func(store.BlockchainStore)) {
	dir, err := ioutil.TempDir("", "BlockchainStore")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	db, err := leveldb.NewLevelDB(dir)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ut(store.NewBlockchainDatabase(db))
}

func Test_Genesis_DefaultGenesis(t *testing.T) {
	default1 := DefaultGenesis(nil)
	default2 := DefaultGenesis(nil)

	assert.Equal(t, default1.header, default2.header)
}

func Test_Genesis_Init_DefaultGenesis(t *testing.T) {
	testBlockchainDatabase(func(bcStore store.BlockchainStore) {
		genesis := DefaultGenesis(bcStore)
		genesisHash := genesis.header.Hash()

		err := genesis.Initialize()
		if err != nil {
			panic(err)
		}

		hash, err := bcStore.GetBlockHash(0)
		assert.Equal(t, err, error(nil))
		assert.Equal(t, hash, genesisHash)

		headHash, err := bcStore.GetHeadBlockHash()
		assert.Equal(t, err, error(nil))
		assert.Equal(t, headHash, genesisHash)
	})
}

func Test_Genesis_Init_GenesisMismatch(t *testing.T) {
	testBlockchainDatabase(func(bcStore store.BlockchainStore) {
		header := DefaultGenesis(bcStore).header.Clone()
		header.Nonce = 38
		bcStore.PutBlockHeader(header, true)

		genesis := DefaultGenesis(bcStore)
		err := genesis.Initialize()
		assert.Equal(t, err, ErrGenesisHashMismatch)
	})
}
