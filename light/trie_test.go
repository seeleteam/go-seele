/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/trie"
	"github.com/stretchr/testify/assert"
)

type mockOdrRetriever struct {
	resp odrResponse
}

func (r *mockOdrRetriever) retrieveWithFilter(request odrRequest, filter peerFilter) (odrResponse, error) {
	return r.resp, nil
}

func Test_Trie_Get(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	// prepare trie on server side
	dbPrefix := []byte("test prefix")
	trie := trie.NewEmptyTrie(dbPrefix, db)
	trie.Put([]byte("hello"), []byte("HELLO"))
	trie.Put([]byte("seele"), []byte("SEELE"))
	trie.Put([]byte("world"), []byte("WORLD"))

	// prepare mock odr retriever
	proof, err := trie.GetProof([]byte("seele"))
	assert.Nil(t, err)
	retriever := &mockOdrRetriever{
		resp: &odrTriePoof{
			Proof: mapToArray(proof),
		},
	}

	// validate on light client
	lightTrie := newOdrTrie(retriever, trie.Hash(), dbPrefix, common.EmptyHash)

	// key exists
	v, ok, err := lightTrie.Get([]byte("seele"))
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, []byte("SEELE"), v)

	// key not found
	v, ok, err = lightTrie.Get([]byte("seele 2"))
	assert.Nil(t, err)
	assert.False(t, ok)
	assert.Nil(t, v)
}
