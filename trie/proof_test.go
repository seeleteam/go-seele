/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package trie

import (
	"bytes"
	crand "crypto/rand"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	common2 "github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
)

func init() {
	mrand.Seed(time.Now().Unix())
}

func TestProof(t *testing.T) {
	trie, vals := randomTrie(500)
	root := trie.Hash()
	for _, kv := range vals {
		proofs, err := trie.Prove(kv.k)
		if err != nil {
			t.Fatalf("missing key %x while constructing proof", kv.k)
		}
		val, err := VerifyProof(root, kv.k, proofs)
		if err != nil {
			t.Fatalf("VerifyProof error for key %x: %v\nraw proof: %v", kv.k, err, proofs)
		}
		if !bytes.Equal(val, kv.v) {
			t.Fatalf("VerifyProof returned wrong value for key %x: got %x, want %x", kv.k, val, kv.v)
		}
	}
}

func TestOneElementProof(t *testing.T) {
	_, trie, dispose := newTestTrie()
	defer dispose()

	trie.Put([]byte("k"), []byte("v"))
	proofs, err := trie.Prove([]byte("k"))
	if err != nil {
		t.Fatal(err)
	}

	if len(proofs) != 1 {
		t.Error("proof should have one element")
	}

	val, err := VerifyProof(trie.Hash(), []byte("k"), proofs)
	if err != nil {
		t.Fatalf("VerifyProof error: %v\n", err)
	}
	if !bytes.Equal(val, []byte("v")) {
		t.Fatalf("VerifyProof returned wrong value: got %x, want 'k'", val)
	}
}

func TestVerifyBadProof(t *testing.T) {
	trie, vals := randomTrie(800)
	root := trie.Hash()
	for _, kv := range vals {
		proofs, err := trie.Prove(kv.k)
		if err != nil {
			t.Fatal(err)
		}

		if len(proofs) == 0 {
			t.Fatal("zero length proof")
		}

		var key string
		for key, _ = range proofs {
			break
		}

		node, _ := proofs[key]
		delete(proofs, key)
		mutateByte(node)
		proofs[string(crypto.HashBytes(node).Bytes())] = node
		if _, err := VerifyProof(root, kv.k, proofs); err == nil {
			t.Fatalf("expected proof to fail for key %x", kv.k)
		}
	}
}

// mutateByte changes one byte in b.
func mutateByte(b []byte) {
	for r := mrand.Intn(len(b)); ; {
		new := byte(mrand.Intn(255))
		if new != b[r] {
			b[r] = new
			break
		}
	}
}

func BenchmarkProve(b *testing.B) {
	trie, vals := randomTrie(100)
	var keys []string
	for k := range vals {
		keys = append(keys, k)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kv := vals[keys[i%len(keys)]]
		proofs, err := trie.Prove(kv.k)
		if err != nil {
			b.Fatal(err)
		}

		if len(proofs) == 0 {
			b.Fatalf("zero length proof for %x", kv.k)
		}
	}
}

func BenchmarkVerifyProof(b *testing.B) {
	trie, vals := randomTrie(100)
	root := trie.Hash()
	var keys []string
	var proofs []map[string][]byte
	for k := range vals {
		keys = append(keys, k)
		proof, err := trie.Prove([]byte(k))
		if err != nil {
			b.Fatal(err)
		}

		proofs = append(proofs, proof)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		im := i % len(keys)
		if _, err := VerifyProof(root, []byte(keys[im]), proofs[im]); err != nil {
			b.Fatalf("key %x: %v", keys[im], err)
		}
	}
}

type kv struct {
	k, v []byte
	t    bool
}

func randomTrie(n int) (*Trie, map[string]*kv) {
	db, _ := leveldb.NewTestDatabase()
	trie, err := NewTrie(common2.EmptyHash, []byte("trietest"), db)
	if err != nil {
		panic(err)
	}

	vals := make(map[string]*kv)
	for i := byte(0); i < 100; i++ {
		value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		value2 := &kv{common.LeftPadBytes([]byte{i + 10}, 32), []byte{i}, false}
		trie.Put(value.k, value.v)
		trie.Put(value2.k, value2.v)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}
	for i := 0; i < n; i++ {
		value := &kv{randBytes(32), randBytes(20), false}
		trie.Put(value.k, value.v)
		vals[string(value.k)] = value
	}
	return trie, vals
}

func randBytes(n int) []byte {
	r := make([]byte, n)
	crand.Read(r)
	return r
}
