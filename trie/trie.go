/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package trie

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto/sha3"
	"github.com/seeleteam/go-seele/database"
)

var (
	errNodeFormat   = errors.New("trie node format is invalid")
	errNodeNotExist = errors.New("trie node not found")
)

// Database is used to load trie nodes by hash.
// It's levelDB in full node, and odrDB in light node.
type Database interface {
	Get(key []byte) ([]byte, error)
}

// Trie is a Merkle Patricia Trie
type Trie struct {
	db       Database
	root     noder     // root node of the Trie
	dbprefix []byte    // db prefix of Trie node
	sha      hash.Hash // hash calc for trie
}

// NewTrie new a trie tree
// param dbprefix will be used as prefix of hash key to save db.
// because we save all of trie trees in the same db,dbprefix protects key/values for different trees
func NewTrie(root common.Hash, dbprefix []byte, db Database) (*Trie, error) {
	trie := NewEmptyTrie(dbprefix, db)

	if !root.IsEmpty() {
		rootnode, err := trie.loadNode(root.Bytes())
		if err != nil {
			return nil, err
		}
		trie.root = rootnode
	}

	return trie, nil
}

// NewEmptyTrie creates an empty trie tree.
func NewEmptyTrie(dbprefix []byte, db Database) *Trie {
	return &Trie{
		db:       db,
		dbprefix: dbprefix,
		sha:      sha3.NewKeccak256(),
	}
}

// Put add or update [key,value] in the trie
func (t *Trie) Put(key, value []byte) error {
	key = keybytesToHex(key)

	node, err := t.insert(t.root, key, value)
	if err == nil {
		t.root = node
	}

	return err
}

// Delete delete node with key in the trie
// return true is delete successfully;false mean the key not exist
func (t *Trie) Delete(key []byte) (bool, error) {
	if t.root == nil {
		return false, nil
	}

	key = keybytesToHex(key)

	match, newnode, err := t.delete(t.root, key, false)
	if err == nil && match {
		t.root = newnode
	}

	return match, err
}

// DeletePrefix deletes nodes with specified prefix in the trie.
// Return true if any node deleted, otherwise false.
// Note, no node deleted if the prefix is nil or empty.
func (t *Trie) DeletePrefix(prefix []byte) (bool, error) {
	if len(prefix) == 0 || t.root == nil {
		return false, nil
	}

	key := keybytesToHex(prefix)
	key = key[0 : len(key)-1] // exclude the terminate key.

	match, newNode, err := t.delete(t.root, key, true)
	if err == nil && match {
		t.root = newNode
	}

	return match, err
}

// Get get the value by key
func (t *Trie) Get(key []byte) ([]byte, bool, error) {
	if t.root == nil {
		return nil, false, nil
	}

	key = keybytesToHex(key)
	val, found, _, err := t.get(t.root, key, 0)

	return val, found, err
}

// Hash return the hash of trie
func (t *Trie) Hash() common.Hash {
	if t.root != nil {
		buf := new(bytes.Buffer)
		t.sha.Reset()
		nodeHash(t.root, buf, t.sha, nil, nil)
		return common.BytesToHash(t.root.Hash())
	}
	return common.EmptyHash
}

// Commit commit the dirty node to database with given batch.
// Note, it will panic on nil batch, please use Hash() instead
// to get the root hash.
func (t *Trie) Commit(batch database.Batch) common.Hash {
	if t.root != nil {
		buf := new(bytes.Buffer)
		t.sha.Reset()
		nodeHash(t.root, buf, t.sha, batch, t.dbprefix)
		return common.BytesToHash(t.root.Hash())
	}
	return common.EmptyHash
}

func nodeHash(node noder, buf *bytes.Buffer, sha hash.Hash, batch database.Batch, dbPrefix []byte) []byte {
	if node == nil {
		return nil
	}

	// node already persisted after call Commit(batch)
	if node.Status() == nodeStatusPersisted {
		return node.Hash()
	}

	// node hash alredy updated after call Hash()
	if node.Status() == nodeStatusUpdated && batch == nil {
		return node.Hash()
	}

	// node hash is dirty or requires to commit with specified batch
	switch n := node.(type) {
	case *LeafNode:
		buf.Reset()
		rlp.Encode(buf, []interface{}{
			n.Key,
			n.Value,
		})
	case *ExtensionNode:
		nexthash := nodeHash(n.NextNode, buf, sha, batch, dbPrefix)

		buf.Reset()
		rlp.Encode(buf, []interface{}{
			true, //add it to diff with extension node;modify later using compact func?
			n.Key,
			nexthash,
		})
	case *BranchNode:
		var children [numBranchChildren][]byte
		for i, child := range n.Children {
			children[i] = nodeHash(child, buf, sha, batch, dbPrefix)
		}

		buf.Reset()
		rlp.Encode(buf, []interface{}{
			children,
		})
	case hashNode:
		return n.Hash()
	default:
		panic(fmt.Sprintf("invalid node: %v", node))
	}

	// update node hash and status if dirty
	if node.Status() == nodeStatusDirty {
		sha.Reset()
		sha.Write(buf.Bytes())
		hash := sha.Sum(nil)
		node.SetHash(hash)
		node.SetStatus(nodeStatusUpdated)
	}

	// persist node if batch specified
	if batch != nil {
		batch.Put(append(dbPrefix, node.Hash()...), buf.Bytes())
		node.SetStatus(nodeStatusPersisted)
	}

	return node.Hash()
}

func encodeNode(node noder, buf *bytes.Buffer, sha hash.Hash) {
	if node == nil {
		return
	}

	// node hash is dirty or requires to commit with specified batch
	switch n := node.(type) {
	case *LeafNode:
		buf.Reset()
		rlp.Encode(buf, []interface{}{
			n.Key,
			n.Value,
		})
	case *ExtensionNode:
		nexthash := nodeHash(n.NextNode, buf, sha, nil, nil)

		buf.Reset()
		rlp.Encode(buf, []interface{}{
			true, //add it to diff with extension node;modify later using compact func?
			n.Key,
			nexthash,
		})
	case *BranchNode:
		var children [numBranchChildren][]byte
		for i, child := range n.Children {
			children[i] = nodeHash(child, buf, sha, nil, nil)
		}

		buf.Reset()
		rlp.Encode(buf, []interface{}{
			children,
		})
	default:
		panic(fmt.Sprintf("invalid node: %v", node))
	}
}

// inserts key-value under the specified node, and returns the new node if changed,
// Otherwise the node itself.
func (t *Trie) insert(node noder, key []byte, value []byte) (noder, error) {
	switch n := node.(type) {
	case *ExtensionNode:
		return t.insertExtensionNode(n, key, value)
	case *LeafNode:
		return t.insertLeafNode(n, key, value)
	case *BranchNode:
		child, err := t.insert(n.Children[key[0]], key[1:], value)
		if err != nil {
			return nil, err
		}
		n.Children[key[0]] = child
		n.status = nodeStatusDirty
		return n, nil
	case hashNode:
		loadnode, err := t.loadNode(n)
		if err != nil {
			return nil, err
		}
		return t.insert(loadnode, key, value)
	case nil:
		return &LeafNode{
			Key:   key,
			Value: value,
		}, nil
	default:
		panic(fmt.Errorf("invalid node: %v", node))
	}
}

func (t *Trie) insertExtensionNode(n *ExtensionNode, key []byte, value []byte) (noder, error) {
	matchlen := matchkeyLen(n.Key, key)

	// key match and insert in nextNode
	if matchlen == len(n.Key) {
		newNode, err := t.insert(n.NextNode, key[matchlen:], value)
		if err != nil {
			return nil, err
		}

		n.NextNode = newNode
		n.status = nodeStatusDirty

		return n, nil
	}

	branchnode := &BranchNode{}

	if matchlen != len(n.Key)-1 {
		branchnode.Children[n.Key[matchlen]] = n
		n.Key = n.Key[matchlen+1:]
		n.status = nodeStatusDirty
	} else {
		branchnode.Children[n.Key[matchlen]] = n.NextNode
	}

	newNode, err := t.insert(nil, key[matchlen+1:], value)
	if err != nil {
		return nil, err
	}

	branchnode.Children[key[matchlen]] = newNode

	// not match key value return branch node
	if matchlen == 0 {
		return branchnode, nil
	}

	// have match key, return extension node
	return &ExtensionNode{
		Key:      key[:matchlen],
		NextNode: branchnode,
	}, nil
}

func (t *Trie) insertLeafNode(n *LeafNode, key []byte, value []byte) (noder, error) {
	matchlen := matchkeyLen(n.Key, key)

	// key match, change the value of leaf node
	if matchlen == len(n.Key) {
		n.Value = value
		n.status = nodeStatusDirty
		return n, nil
	}

	branchnode := &BranchNode{}
	branchnode.Children[n.Key[matchlen]] = n
	n.Key = n.Key[matchlen+1:]
	n.status = nodeStatusDirty

	newNode, err := t.insert(nil, key[matchlen+1:], value)
	if err != nil {
		return nil, err
	}
	branchnode.Children[key[matchlen]] = newNode

	// not match key value return branch node
	if matchlen == 0 {
		return branchnode, nil
	}

	// have match key, return extension node
	return &ExtensionNode{
		Key:      key[:matchlen],
		NextNode: branchnode,
	}, nil
}

func (t *Trie) delete(node noder, key []byte, descendant bool) (bool, noder, error) {
	switch n := node.(type) {
	case *LeafNode:
		matchlen := matchkeyLen(key, n.Key)
		if matchlen == len(n.Key) || matchlen == len(key) {
			return true, nil, nil
		}

		return false, n, nil
	case *ExtensionNode:
		matchlen := matchkeyLen(key, n.Key)
		if descendant && matchlen == len(key) {
			return true, nil, nil
		}

		if matchlen == len(n.Key) {
			match, newnode, err := t.delete(n.NextNode, key[matchlen:], descendant)
			if err != nil {
				return false, nil, err
			}

			if match {
				n.status = nodeStatusDirty
				n.NextNode = newnode
				if newnode == nil {
					return true, nil, nil
				}
				return true, n, nil
			}
		}

		return false, n, nil
	case *BranchNode:
		if descendant && len(key) == 1 {
			if n.Children[key[0]] == nil {
				return false, n, nil
			}

			n.Children[key[0]] = nil
		} else {
			match, newnode, err := t.delete(n.Children[key[0]], key[1:], descendant)
			if err != nil {
				return false, nil, err
			}

			if !match {
				return false, n, nil
			}

			n.Children[key[0]] = newnode
		}

		n.status = nodeStatusDirty

		pos := -1
		count := 0
		for i, child := range n.Children {
			if child != nil {
				pos = i
				count++
			}
		}

		if count == 1 {
			var childnode noder
			var err error
			childnode = n.Children[pos]
			if hashnode, ok := childnode.(hashNode); ok {
				childnode, err = t.loadNode(hashnode)
				if err != nil {
					return true, nil, err
				}
			}
			switch childnode := childnode.(type) {
			case *LeafNode:
				newnode := &LeafNode{
					Key:   append([]byte{byte(pos)}, childnode.Key...),
					Value: childnode.Value,
				}
				return true, newnode, nil
			case *ExtensionNode:
				newnode := &ExtensionNode{
					Key:      append([]byte{byte(pos)}, childnode.Key...),
					NextNode: childnode.NextNode,
				}
				return true, newnode, nil
			}
		}
		return true, n, nil
	case hashNode:
		loadnode, err := t.loadNode(n)
		if err != nil {
			return false, nil, err
		}
		return t.delete(loadnode, key, descendant)
	case nil:
		return false, nil, nil
	default:
		panic(fmt.Sprintf("invalid node: %v (%v)", n, key))
	}
}

// loadNode get node from memory cache or database
func (t *Trie) loadNode(hash []byte) (noder, error) {
	//TODO need cache nodes
	key := append(t.dbprefix, hash...)
	val, err := t.db.Get(key)
	if err != nil || len(val) == 0 {
		return nil, errNodeNotExist
	}
	return decodeNode(hash, val)
}

// decodeNode decode node from buf byte
func decodeNode(hash, value []byte) (noder, error) {
	if len(value) == 0 {
		return nil, io.ErrUnexpectedEOF
	}

	vals, _, err := rlp.SplitList(value)
	if err != nil {
		return nil, err
	}

	n, err := rlp.CountValues(vals)
	if err != nil {
		return nil, err
	}

	switch n {
	case 1:
		return decodeBranchNode(hash, vals)
	case 2:
		return decodeLeafNode(hash, vals)
	case 3:
		return decodeExtensionNode(hash, vals)
	default:
		return nil, errNodeFormat
	}
}

func decodeLeafNode(hash, values []byte) (noder, error) {
	key, rest, err := rlp.SplitString(values)
	if err != nil {
		return nil, err
	}

	val, _, err := rlp.SplitString(rest)
	if err != nil {
		return nil, err
	}

	return &LeafNode{
		Node:  newPersistedNode(hash),
		Key:   key,
		Value: val,
	}, nil
}

func decodeExtensionNode(hash, values []byte) (noder, error) {
	_, bufs, err := rlp.SplitString(values)
	if err != nil {
		return nil, err
	}

	key, rest, err := rlp.SplitString(bufs)
	if err != nil {
		return nil, err
	}

	val, _, err := rlp.SplitString(rest)
	if err != nil {
		return nil, err
	}

	return &ExtensionNode{
		Node:     newPersistedNode(hash),
		Key:      key,
		NextNode: append(hashNode{}, val...),
	}, nil
}

func decodeBranchNode(hash, values []byte) (noder, error) {
	kind, elems, _, err := rlp.Split(values)
	if err != nil {
		return nil, err
	}

	itemcount, err := rlp.CountValues(elems)
	if err != nil {
		return nil, err
	}

	if kind != rlp.List && itemcount != numBranchChildren {
		return nil, errNodeFormat
	}

	branchnode := &BranchNode{
		Node: newPersistedNode(hash),
	}

	for i := 0; i < numBranchChildren; i++ {
		kind, val, rest, err := rlp.Split(elems)
		if err != nil {
			return nil, err
		}

		elems = rest

		if kind == rlp.String {
			if length := len(val); length == common.HashLength {
				branchnode.Children[i] = append(hashNode{}, val...)
			} else {
				branchnode.Children[i] = nil
			}
		}
	}

	return branchnode, nil
}

func (t *Trie) get(node noder, key []byte, pos int) ([]byte, bool, noder, error) {
	switch n := (node).(type) {
	case nil:
		return nil, false, nil, nil
	case *ExtensionNode:
		if len(key)-pos < len(n.Key) || !bytes.Equal(n.Key, key[pos:pos+len(n.Key)]) {
			return nil, false, n, nil
		}
		val, found, newnode, err := t.get(n.NextNode, key, pos+len(n.Key))
		n.NextNode = newnode
		return val, found, n, err
	case hashNode:
		child, err := t.loadNode(n)
		if err != nil {
			return nil, false, n, err
		}
		return t.get(child, key, pos)
	case *LeafNode:
		if len(key)-pos < len(n.Key) || !bytes.Equal(n.Key, key[pos:pos+len(n.Key)]) {
			// key not found in trie
			return nil, false, n, nil
		}
		return n.Value, true, n, nil
	case *BranchNode:
		val, found, newnode, err := t.get(n.Children[key[pos]], key, pos+1)
		n.Children[key[pos]] = newnode
		return val, found, n, err
	default:
		panic(fmt.Sprintf("invalid node: %v", node))
	}
}

func keybytesToHex(str []byte) []byte {
	l := len(str)*2 + 1
	var nibbles = make([]byte, l)
	for i, b := range str {
		nibbles[i*2] = b / byte(numBranchChildren-1)   // now is b / 16
		nibbles[i*2+1] = b % byte(numBranchChildren-1) // now is b% 16
	}
	nibbles[l-1] = byte(numBranchChildren - 1) // term key is 16
	return nibbles
}

func matchkeyLen(a, b []byte) int {
	length := len(a)
	lengthb := len(b)
	if lengthb < length {
		length = lengthb
	}
	var i = 0
	for ; i < length; i++ {
		if a[i] != b[i] {
			break
		}
	}
	return i
}
