/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package trie

import (
	"fmt"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func newTestTrie() (database.Database, *Trie, func()) {
	db, dispose := leveldb.NewTestDatabase()
	trie, err := NewTrie(common.EmptyHash, []byte("trietest"), db)
	if err != nil {
		dispose()
		panic(err)
	}

	return db, trie, dispose
}

func trieMustGet(trie *Trie, key []byte) ([]byte, bool) {
	val, found, err := trie.Get(key)
	if err != nil {
		panic("trie.Get failed")
	}

	return val, found
}

func Test_trie_Update(t *testing.T) {
	db, trie, remove := newTestTrie()
	defer remove()

	trie.Put([]byte("12345678"), []byte("test"))
	trie.Put([]byte("12345678"), []byte("testnew"))
	trie.Put([]byte("12345557"), []byte("test1"))
	trie.Put([]byte("12375879"), []byte("test2"))
	trie.Put([]byte("02375879"), []byte("test3"))
	trie.Put([]byte("04375879"), []byte("test4"))
	trie.Put([]byte("24375879"), []byte("test5"))
	trie.Put([]byte("24375878"), []byte("test6"))
	trie.Put([]byte("24355879"), []byte("test7"))
	value, _ := trieMustGet(trie, []byte("12345678"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "testnew")
	value, _ = trieMustGet(trie, []byte("12345557"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test1")
	value, _ = trieMustGet(trie, []byte("12375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test2")
	value, _ = trieMustGet(trie, []byte("02375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test3")
	value, _ = trieMustGet(trie, []byte("04375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test4")
	value, _ = trieMustGet(trie, []byte("24375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test5")
	value, _ = trieMustGet(trie, []byte("24375878"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test6")
	value, _ = trieMustGet(trie, []byte("24355879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test7")
	batch := db.NewBatch()
	trie.Commit(batch)
	assert.Equal(t, batch.Commit(), nil)
}

func trieMustDelete(trie *Trie, key []byte) bool {
	deleted, err := trie.Delete(key)
	if err != nil {
		panic("trie.Delete failed")
	}

	return deleted
}

func Test_trie_Delete(t *testing.T) {
	_, trie, remove := newTestTrie()
	defer remove()

	trie.Put([]byte("12345678123"), []byte("test"))
	trie.Put([]byte("12345557"), []byte("test1"))
	trie.Put([]byte("12375879321"), []byte("test2"))
	trie.Put([]byte("02375879"), []byte("test3"))
	trie.Put([]byte("04375879"), []byte("test4"))
	trie.Put([]byte("24375879"), []byte("test5"))
	trie.Put([]byte("24375878"), []byte("test6"))
	trie.Put([]byte("24355879"), []byte("test7"))
	match := trieMustDelete(trie, []byte("12345678123"))
	fmt.Println(match)
	assert.Equal(t, match, true)
	match = trieMustDelete(trie, []byte("12375879321"))
	fmt.Println(match)
	assert.Equal(t, match, true)
	match = trieMustDelete(trie, []byte("24375879"))
	fmt.Println(match)
	assert.Equal(t, match, true)
	match = trieMustDelete(trie, []byte("24375889"))
	fmt.Println(match)
	assert.Equal(t, match, false)
	value, found := trieMustGet(trie, []byte("12345678123"))
	fmt.Println(string(value))
	assert.False(t, found)
	value, found = trieMustGet(trie, []byte("12375879321"))
	fmt.Println(string(value))
	assert.False(t, found)
	value, _ = trieMustGet(trie, []byte("02375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test3")
	value, _ = trieMustGet(trie, []byte("04375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test4")
	value, found = trieMustGet(trie, []byte("24375879"))
	fmt.Println(string(value))
	assert.False(t, found)
	value, _ = trieMustGet(trie, []byte("24375878"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test6")
	value, _ = trieMustGet(trie, []byte("24355879"))
	assert.Equal(t, string(value), "test7")
	fmt.Println(string(value))
	fmt.Println(trie.Hash())
}

func Test_trie_Commit(t *testing.T) {
	db, trie, remove := newTestTrie()
	defer remove()

	trie.Put([]byte("12345678"), []byte("test"))
	trie.Put([]byte("12345557"), []byte("test1"))
	trie.Put([]byte("12375879"), []byte("test2"))
	trie.Put([]byte("02375879"), []byte("test3"))
	trie.Put([]byte("04375879"), []byte("test4"))
	trie.Put([]byte("24375879"), []byte("test5"))
	trie.Put([]byte("24375878"), []byte("test6"))
	trie.Put([]byte("24355879"), []byte("test7"))

	batch := db.NewBatch()
	hash := trie.Commit(batch)
	batch.Commit()
	fmt.Println(hash)

	fmt.Println(string("----------------------------------"))
	trienew, err := NewTrie(hash, []byte("trietest"), db)
	assert.Equal(t, err, nil)

	trienew.Delete([]byte("24355879"))
	trienew.Put([]byte("243558790"), []byte("test8"))
	trienew.Put([]byte("043758790"), []byte("test9"))

	value, _ := trieMustGet(trienew, []byte("12345678"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test")
	value, _ = trieMustGet(trienew, []byte("12345557"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test1")
	value, _ = trieMustGet(trienew, []byte("12375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test2")
	value, _ = trieMustGet(trienew, []byte("02375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test3")
	value, _ = trieMustGet(trienew, []byte("04375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test4")
	value, _ = trieMustGet(trienew, []byte("24375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test5")
	value, _ = trieMustGet(trienew, []byte("24375878"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test6")
	value, found := trieMustGet(trienew, []byte("24355879"))
	fmt.Println(string(value))
	assert.False(t, found)
	value, _ = trieMustGet(trienew, []byte("243558790"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test8")
	value, _ = trieMustGet(trienew, []byte("043758790"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test9")
	value, _ = trieMustGet(trienew, []byte("12345557"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test1")
	value, _ = trieMustGet(trienew, []byte("12375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test2")
}

func Test_trie_CommitOneByOne(t *testing.T) {
	db, trie, remove := newTestTrie()
	defer remove()

	trie.Put([]byte{1, 2, 3}, []byte{1, 2, 3})
	trie.Hash()
	trie.Put([]byte{1, 2, 4}, []byte{1, 2, 4})
	trie.Hash()

	batch := db.NewBatch()
	hash := trie.Commit(batch)
	batch.Commit()

	trienew, err := NewTrie(hash, []byte("trietest"), db)
	if err != nil {
		panic(err)
	}

	value, _ := trieMustGet(trienew, []byte{1, 2, 3})
	assert.Equal(t, value, []byte{1, 2, 3})

	value, _ = trieMustGet(trienew, []byte{1, 2, 4})
	assert.Equal(t, value, []byte{1, 2, 4})
}

const benchElemCount = 20000

var addrList [][]byte
var code = make([]byte, 4*1024, 4*1024) // 4KB bytes code size

func init() {
	for i := 0; i < benchElemCount; i++ {
		addr := *crypto.MustGenerateRandomAddress()
		addrList = append(addrList, addr[:])
	}
}

func prepareData(trie *Trie) {
	for _, addr := range addrList {
		if err := trie.Put(addr, code); err != nil {
			panic(err)
		}
	}
}

func Benchmark_Trie_Get(b *testing.B) {
	_, trie, dispose := newTestTrie()
	defer dispose()

	prepareData(trie)
	key := addrList[len(addrList)/2]
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, found := trieMustGet(trie, key); !found {
			panic("value not found by key")
		}
	}
}

func Benchmark_Trie_Put(b *testing.B) {
	_, trie, dispose := newTestTrie()
	defer dispose()

	for i, addrLen := 0, len(addrList); i < b.N; i++ {
		if err := trie.Put(addrList[i%addrLen], code); err != nil {
			panic(err)
		}
	}
}

func Benchmark_Trie_Commit(b *testing.B) {
	_, trie, dispose := newTestTrie()
	defer dispose()

	prepareData(trie)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		trie.Commit(nil)
	}
}

func Test_Trie_Delete_SingleRoot(t *testing.T) {
	_, trie, remove := newTestTrie()
	defer remove()

	// insert a leaf node as root
	assert.Equal(t, trie.Put([]byte{1, 2, 3}, []byte("value")), nil)

	// key mismatch
	assert.Equal(t, trieMustDelete(trie, []byte{1, 2}), false)       // less keys
	assert.Equal(t, trieMustDelete(trie, []byte{1, 2, 4}), false)    // same keys with invalid value
	assert.Equal(t, trieMustDelete(trie, []byte{1, 2, 3, 4}), false) // more keys

	// key match
	assert.Equal(t, trieMustDelete(trie, []byte{1, 2, 3}), true)
	assert.Equal(t, trie.root, nil)
}

func Test_Trie_Delete_ExtNode(t *testing.T) {
	_, trie, remove := newTestTrie()
	defer remove()

	// ext node with key 1,2
	assert.Equal(t, trie.Put([]byte{1, 2, 3}, []byte("1")), nil)
	assert.Equal(t, trie.Put([]byte{1, 2, 4}, []byte("2")), nil)

	// cannot delete with prefix 1,2
	assert.Equal(t, trieMustDelete(trie, []byte{1, 2}), false)
}

func Test_Trie_Delete_Branch(t *testing.T) {
	_, trie, remove := newTestTrie()
	defer remove()

	// insert nodes to construct a branch node
	assert.Equal(t, trie.Put([]byte{1, 2, 0x36}, []byte("1")), nil) // branch.children[3]
	assert.Equal(t, trie.Put([]byte{1, 2, 0x68}, []byte("2")), nil) // branch.children[6]
	assert.Equal(t, trie.Put([]byte{1, 2}, []byte("3")), nil)       // branch.children[16] (terminate)

	// key mismatch
	assert.Equal(t, trieMustDelete(trie, []byte{1, 2, 0x46}), false) // branch.children[4]
	assert.Equal(t, trieMustDelete(trie, []byte{1, 2, 0x35}), false) // branch.children[3] but with invalid value

	// key match
	assert.Equal(t, trieMustDelete(trie, []byte{1, 2, 0x36}), true)
	assert.Equal(t, trieMustDelete(trie, []byte{1, 2, 0x68}), true)
	assert.Equal(t, trieMustDelete(trie, []byte{1, 2}), true)
	assert.Equal(t, trie.root, nil)
}

func trieMustDeletePrefix(trie *Trie, prefix []byte) bool {
	deleted, err := trie.DeletePrefix(prefix)
	if err != nil {
		panic("trie.DeletePrefix failed")
	}

	return deleted
}

func Test_Trie_DeletePrefix_EmptyKey(t *testing.T) {
	_, trie, remove := newTestTrie()
	defer remove()

	assert.Equal(t, trie.Put([]byte{1, 2, 3}, []byte("v")), nil)

	assert.Equal(t, trieMustDeletePrefix(trie, nil), false)
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{}), false)
}

func Test_Trie_DeletePrefix_LeafNode(t *testing.T) {
	_, trie, remove := newTestTrie()
	defer remove()

	// leaf node with key 1,2,3
	assert.Equal(t, trie.Put([]byte{1, 2, 3}, []byte("v")), nil)

	// key mismatch
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{1, 2, 3, 4}), false) // more keys
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{1, 2, 4}), false)    // same keys with invalid value
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{2}), false)          // less keys with invalid value

	// exact keys
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{1, 2, 3}), true)
	assert.Equal(t, trie.root, nil)

	// less keys
	assert.Equal(t, trie.Put([]byte{1, 2, 3}, []byte("v")), nil)
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{1, 2}), true)
	assert.Equal(t, trie.root, nil)
}

func Test_Trie_DeletePrefix_ExtNode(t *testing.T) {
	_, trie, remove := newTestTrie()
	defer remove()

	// ext node with key 1,2
	assert.Equal(t, trie.Put([]byte{1, 2, 3}, []byte("1")), nil)
	assert.Equal(t, trie.Put([]byte{1, 2, 4}, []byte("2")), nil)

	// key mismatch
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{1, 3}), false) // same keys with invalid value
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{2}), false)    // less keys with invalid value

	// exact keys
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{1, 2}), true)
	assert.Equal(t, trie.root, nil)

	// less keys
	assert.Equal(t, trie.Put([]byte{1, 2, 3}, []byte("1")), nil)
	assert.Equal(t, trie.Put([]byte{1, 2, 4}, []byte("2")), nil)
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{1}), true)
	assert.Equal(t, trie.root, nil)
}

func Test_Trie_DeletePrefix_BranchNode(t *testing.T) {
	_, trie, remove := newTestTrie()
	defer remove()

	// branch node
	assert.Equal(t, trie.Put([]byte{1, 2, 3, 5}, []byte("1")), nil) // branch.children[3]
	assert.Equal(t, trie.Put([]byte{1, 2, 4, 6}, []byte("2")), nil) // branch.children[4]

	// key mismatch
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{1, 2, 5}), false) // branch.children[5]

	// key match
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{1, 2, 3}), true) // branch.children[3]
	assert.Equal(t, trieMustDeletePrefix(trie, []byte{1, 2, 4}), true) // leaf node
	assert.Equal(t, trie.root, nil)
}
