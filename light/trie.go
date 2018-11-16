/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/trie"
)

type odrDatabase struct {
	kvs map[string][]byte
}

func newOdrDatabase() *odrDatabase {
	return &odrDatabase{make(map[string][]byte)}
}

// Get implements the trie.Database interface to store trie node key-value pairs.
func (db *odrDatabase) Get(key []byte) ([]byte, error) {
	return db.kvs[string(key)], nil
}

type odrRetriever interface {
	retrieveWithFilter(request odrRequest, filter peerFilter) (odrResponse, error)
}

type odrTrie struct {
	odr       odrRetriever
	root      common.Hash
	db        *odrDatabase
	dbPrefix  []byte
	trie      *trie.Trie
	blockHash common.Hash
}

func newOdrTrie(retriever odrRetriever, root common.Hash, dbPrefix []byte, blockHash common.Hash) *odrTrie {
	return &odrTrie{
		odr:       retriever,
		root:      root,
		db:        newOdrDatabase(),
		dbPrefix:  dbPrefix,
		blockHash: blockHash,
	}
}

func (t *odrTrie) Hash() common.Hash {
	panic("unsupported")
}

func (t *odrTrie) Commit(batch database.Batch) common.Hash {
	panic("unsupported")
}

func (t *odrTrie) Get(key []byte) ([]byte, bool, error) {
	request := &odrTriePoof{
		Root: t.root,
		Key:  key,
	}

	// send ODR request to get trie proof.
	filter := peerFilter{blockHash: t.blockHash}
	response, err := t.odr.retrieveWithFilter(request, filter)
	if err != nil {
		return nil, false, errors.NewStackedError(err, "failed to retrieve ODR trie proof")
	}

	// insert the trie proof in databse.
	for _, n := range response.(*odrTriePoof).Proof {
		key := append(t.dbPrefix, []byte(n.Key)...)
		t.db.kvs[string(key)] = n.Value
	}

	// construct the MPT for the first time.
	if t.trie == nil {
		t.trie, err = trie.NewTrie(t.root, t.dbPrefix, t.db)
		if err != nil {
			return nil, false, errors.NewStackedError(err, "failed to create trie")
		}
	}

	return t.trie.Get(key)
}

func (t *odrTrie) Put(key, value []byte) error {
	panic("unsupported")
}

func (t *odrTrie) DeletePrefix(prefix []byte) (bool, error) {
	panic("unsupported")
}

func (t *odrTrie) GetProof(key []byte) (map[string][]byte, error) {
	panic("unsupported")
}
