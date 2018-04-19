package trie

import (
	"fmt"
	"testing"

	"github.com/seeleteam/go-seele/common"
)

func TestIterator(t *testing.T) {
	db, remove := newTestTrieDB()
	defer remove()
	trie, err := NewTrie(common.Hash{}, []byte("trietest"), db)
	if err != nil {
		panic(err)
	}

	kvs := []struct{ k, v string } {
		{"do", "verb"},
		{"bombad", "jedi"},
		{"horse", "stallion"},
		{"downfall", "droid"},
		{"doge", "coin"},
		{"rising", "malevolence"},
		{"when surrounded by war", "one must eventually choose a side"},
	}
	all := make(map[string]string)
	for _, kv := range kvs {
		all[kv.k] = kv.v
		trie.Put([]byte(kv.k), []byte(kv.v))
	}

	found := make(map[string]string)
	it := NewIterator(trie.NodeIterator(nil))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}
	fmt.Println(found)
	for k, v := range all {
		if found[k] != v {
			t.Errorf("iterator value mismatch for %s: got %q want %q", k, found[k], v)
		}
	}
}
