/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package trie

import (
	"fmt"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
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
	value, _ := trie.Get([]byte("12345678"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "testnew")
	value, _ = trie.Get([]byte("12345557"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test1")
	value, _ = trie.Get([]byte("12375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test2")
	value, _ = trie.Get([]byte("02375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test3")
	value, _ = trie.Get([]byte("04375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test4")
	value, _ = trie.Get([]byte("24375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test5")
	value, _ = trie.Get([]byte("24375878"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test6")
	value, _ = trie.Get([]byte("24355879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test7")
	batch := db.NewBatch()
	trie.Commit(batch)
	assert.Equal(t, batch.Commit(), nil)
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
	match := trie.Delete([]byte("12345678123"))
	fmt.Println(match)
	assert.Equal(t, match, true)
	match = trie.Delete([]byte("12375879321"))
	fmt.Println(match)
	assert.Equal(t, match, true)
	match = trie.Delete([]byte("24375879"))
	fmt.Println(match)
	assert.Equal(t, match, true)
	match = trie.Delete([]byte("24375889"))
	fmt.Println(match)
	assert.Equal(t, match, false)
	value, _ := trie.Get([]byte("12345678123"))
	fmt.Println(string(value))
	assert.Equal(t, len(value), 0)
	value, _ = trie.Get([]byte("12375879321"))
	fmt.Println(string(value))
	assert.Equal(t, len(value), 0)
	value, _ = trie.Get([]byte("02375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test3")
	value, _ = trie.Get([]byte("04375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test4")
	value, _ = trie.Get([]byte("24375879"))
	fmt.Println(string(value))
	assert.Equal(t, len(value), 0)
	value, _ = trie.Get([]byte("24375878"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test6")
	value, _ = trie.Get([]byte("24355879"))
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

	value, _ := trienew.Get([]byte("12345678"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test")
	value, _ = trienew.Get([]byte("12345557"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test1")
	value, _ = trienew.Get([]byte("12375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test2")
	value, _ = trienew.Get([]byte("02375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test3")
	value, _ = trienew.Get([]byte("04375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test4")
	value, _ = trienew.Get([]byte("24375879"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test5")
	value, _ = trienew.Get([]byte("24375878"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test6")
	value, _ = trienew.Get([]byte("24355879"))
	fmt.Println(string(value))
	assert.Equal(t, len(value), 0)
	value, _ = trienew.Get([]byte("243558790"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test8")
	value, _ = trienew.Get([]byte("043758790"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test9")
	value, _ = trienew.Get([]byte("12345557"))
	fmt.Println(string(value))
	assert.Equal(t, string(value), "test1")
	value, _ = trienew.Get([]byte("12375879"))
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

	value, _ := trienew.Get([]byte{1, 2, 3})
	assert.Equal(t, value, []byte{1, 2, 3})

	value, _ = trienew.Get([]byte{1, 2, 4})
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
		if _, found := trie.Get(key); !found {
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
