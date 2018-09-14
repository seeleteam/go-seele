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

// GetProof constructs a merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root node), ending
// with the node that proves the absence of the key.
func (t *Trie) GetProof(key []byte) (map[string][]byte, error) {
	// Collect all nodes on the path to key.
	key = keybytesToHex(key)
	nodes := make([]noder, 0)
	tn := t.root
	proof := make(map[string][]byte)

	for len(key) > 0 && tn != nil {
		switch n := tn.(type) {
		case *ExtensionNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				// The trie doesn't contain the key.
				tn = nil
			} else {
				tn = n.NextNode

				// for ExtensionNode, skip the prefix with len(n.Key),
				key = key[len(n.Key):]
			}
			nodes = append(nodes, n)
		case *BranchNode:
			tn = n.Children[key[0]]

			// for BranchNode, just skip one prefix char,
			key = key[1:]
			nodes = append(nodes, n)
		case hashNode:
			var err error
			tn, err = t.loadNode(n)
			if err != nil {
				return proof, fmt.Errorf("unhandled trie error: %s", err)
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
		hash := nodeHash(n, buf, sha, nil, nil)
		encodeNode(n, buf, sha)

		proof[string(hash)] = buf.Bytes()
	}

	return proof, nil
}

// VerifyProof checks merkle proofs. The given proof must contain the value for
// key in a trie with the given root hash. VerifyProof returns an error if the
// proof contains invalid trie nodes or the wrong value.
func VerifyProof(rootHash common.Hash, key []byte, proof map[string][]byte) (value []byte, err error) {
	key = keybytesToHex(key)
	wantHash := rootHash
	buf := new(bytes.Buffer)
	sha := sha3.NewKeccak256()
	for i := 0; ; i++ {
		encoded := proof[string(wantHash[:])]
		if encoded == nil {
			return nil, fmt.Errorf("proof node %d (hash %064x) missing", i, wantHash)
		}

		n, err := decodeNode(common.CopyBytes(wantHash.Bytes()), encoded)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %d: %v", i, err)
		}

		// verify node hash against proof key to avoid faked node.
		n.SetStatus(nodeStatusDirty)
		if h := nodeHash(n, buf, sha, nil, nil); !bytes.Equal(wantHash.Bytes(), h) {
			return nil, fmt.Errorf("proof node %d hash mismatch", i)
		}

		keyrest, cld := get(n, key)
		switch cld := cld.(type) {
		case nil:
			// The trie doesn't contain the key.
			return nil, nil
		case hashNode:
			key = keyrest
			copy(wantHash[:], cld)
		case *LeafNode:
			return cld.Value, nil
		}
	}
}

func get(tn noder, key []byte) ([]byte, noder) {
	for {
		if len(key) == 0 {
			return nil, nil
		}

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
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				return nil, nil
			}

			return nil, n
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}
}
