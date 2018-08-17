/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package trie

import (
	"bytes"
	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto/sha3"
)

// Putter wraps the database write operation supported by both batches and regular databases.
type Putter interface {
	Put(key []byte, value []byte) error
}

// DatabaseReader wraps the Get and Has method of a backing store for the trie.
type Reader interface {
	// Get retrieves the value associated with key form the database.
	Get(key []byte) (value []byte, err error)

	// Has retrieves whether a key is present in the database.
	Has(key []byte) (bool, error)
}

// Prove constructs a merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root node), ending
// with the node that proves the absence of the key.
func (t *Trie) Prove(key []byte, proofDb Putter) error {
	// Collect all nodes on the path to key.
	key = keybytesToHex(key)
	nodes := make([]noder, 0)
	tn := t.root
	for len(key) > 0 && tn != nil {
		switch n := tn.(type) {
		case *ExtensionNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				// The trie doesn't contain the key.
				tn = nil
			} else {
				tn = n.NextNode
				key = key[len(n.Key):]
			}
			nodes = append(nodes, n)
		case *BranchNode:
			tn = n.Children[key[0]]
			key = key[1:]
			nodes = append(nodes, n)
		case hashNode:
			var err error
			tn, err = t.loadNode(n)
			if err != nil {
				return fmt.Errorf("unhandled trie error: %s", err)
			}
		case *LeafNode:
			tn = nil
			if len(key) >= len(n.Key) && bytes.Equal(n.Key, key[:len(n.Key)]) {
				nodes = append(nodes, n)
			}
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}

	for _, n := range nodes {
		buf := new(bytes.Buffer)
		var sha = sha3.NewKeccak256()
		sha.Reset()
		hash := t.hash(n, buf, sha, nil)
		t.EncodeNode(n, buf, sha)

		proofDb.Put(hash, buf.Bytes())
	}

	return nil
}

// VerifyProof checks merkle proofs. The given proof must contain the value for
// key in a trie with the given root hash. VerifyProof returns an error if the
// proof contains invalid trie nodes or the wrong value.
func VerifyProof(rootHash common.Hash, key []byte, proofDb Reader) (value []byte, err error, nodes int) {
	key = keybytesToHex(key)
	wantHash := rootHash
	for i := 0; ; i++ {
		buf, _ := proofDb.Get(wantHash[:])
		if buf == nil {
			return nil, fmt.Errorf("proof node %d (hash %064x) missing", i, wantHash), i
		}
		n, err := decodeNode(wantHash[:], buf)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %d: %v", i, err), i
		}
		keyrest, cld := get(n, key)
		switch cld := cld.(type) {
		case nil:
			// The trie doesn't contain the key.
			return nil, nil, i
		case hashNode:
			key = keyrest
			copy(wantHash[:], cld)
		case *LeafNode:
			return cld.Value, nil, i + 1
		}
	}
}

func get(tn noder, key []byte) ([]byte, noder) {
	for {
		switch n := tn.(type) {
		case *ExtensionNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				return nil, nil
			}
			tn = n.NextNode
			key = key[len(n.Key):]
		case *BranchNode:
			tn = n.Children[key[0]]
			key = key[1:]
		case hashNode:
			return key, n
		case nil:
			return key, nil
		case *LeafNode:
			return nil, n
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}
}
