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
	errNodeFormat   = errors.New("node format is invalid")
	errNodeNotExist = errors.New("node not exist in db")
)

// Trie is a Merkle Patricia Trie
type Trie struct {
	db       database.Database
	root     noder     // root node of the Trie
	dbprefix []byte    // db prefix of Trie node
	sha      hash.Hash // hash calc for trie
}

// ShallowCopy returns a new trie with the same root.
func (t *Trie) ShallowCopy() (*Trie, error) {
	rootHash := t.Hash()
	t, err := NewTrie(rootHash, t.dbprefix, t.db)
	if err != nil {
		return t, fmt.Errorf("request hash: %s, error: %s", rootHash.ToHex(), err)
	}

	return t, nil
}

// NewTrie new a trie tree
// param dbprefix will be used as prefix of hash key to save db.
// because we save all of trie trees in the same db,dbprefix protects key/values for different trees
func NewTrie(root common.Hash, dbprefix []byte, db database.Database) (*Trie, error) {
	trie := &Trie{
		db:       db,
		dbprefix: dbprefix,
		sha:      sha3.NewKeccak256(),
	}

	if root != common.EmptyHash {
		rootnode, err := trie.loadNode(root.Bytes())
		if err != nil {
			return nil, err
		}
		trie.root = rootnode
	}

	return trie, nil
}

// Put add or update [key,value] in the trie
func (t *Trie) Put(key, value []byte) error {
	key = keybytesToHex(key)
	_, node, err := t.insert(t.root, key, value)
	if err == nil {
		t.root = node
	}
	return err
}

// Delete delete node with key in the trie
// return true is delete successfully;false mean the key not exist
func (t *Trie) Delete(key []byte) bool {
	if t.root != nil {
		key = keybytesToHex(key)
		match, newnode, err := t.delete(t.root, key, false)
		if err == nil && match {
			t.root = newnode
		}
		return match
	}
	return false
}

// DeletePrefix deletes nodes with specified prefix in the trie.
// Return true if any node deleted, otherwise false.
// Note, no node deleted if the prefix is nil or empty.
func (t *Trie) DeletePrefix(prefix []byte) bool {
	if len(prefix) == 0 || t.root == nil {
		return false
	}

	key := keybytesToHex(prefix)
	key = key[0 : len(key)-1] // exclude the terminate key.

	match, newNode, err := t.delete(t.root, key, true)
	if err == nil && match {
		t.root = newNode
	}

	return match
}

// Get get the value by key
func (t *Trie) Get(key []byte) ([]byte, bool) {
	if t.root != nil {
		key = keybytesToHex(key)
		val, _ := t.get(t.root, key, 0)
		if len(val) > 0 {
			return val, true
		}
	}
	return nil, false
}

// Hash return the hash of trie
func (t *Trie) Hash() common.Hash {
	if t.root != nil {
		buf := new(bytes.Buffer)
		t.sha.Reset()
		t.hash(t.root, buf, t.sha, nil)
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
		t.hash(t.root, buf, t.sha, batch)
		return common.BytesToHash(t.root.Hash())
	}
	return common.EmptyHash
}

func (t *Trie) hash(node noder, buf *bytes.Buffer, sha hash.Hash, batch database.Batch) []byte {
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
		nexthash := t.hash(n.NextNode, buf, sha, batch)

		buf.Reset()
		rlp.Encode(buf, []interface{}{
			true, //add it to diff with extension node;modify later using compact func?
			n.Key,
			nexthash,
		})
	case *BranchNode:
		var children [numBranchChildren][]byte
		for i, child := range n.Children {
			children[i] = t.hash(child, buf, sha, batch)
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
		batch.Put(append(t.dbprefix, node.Hash()...), buf.Bytes())
		node.SetStatus(nodeStatusPersisted)
	}

	return node.Hash()
}

// return true if insert succeed,it also mean node is dirty,should recalc hash
func (t *Trie) insert(node noder, key []byte, value []byte) (bool, noder, error) {
	switch n := node.(type) {
	case *ExtensionNode:
		return t.insertExtensionNode(n, key, value)
	case *LeafNode:
		return t.insertLeafNode(n, key, value)
	case *BranchNode:
		_, child, err := t.insert(n.Children[key[0]], key[1:], value)
		if err != nil {
			return false, nil, err
		}
		n.Children[key[0]] = child
		n.status = nodeStatusDirty
		return true, n, nil
	case hashNode:
		loadnode, err := t.loadNode(n)
		if err != nil {
			return false, nil, err
		}
		dirty, newnode, err := t.insert(loadnode, key, value)
		return dirty, newnode, err
	case nil:
		newnode := &LeafNode{
			Node: Node{
				status: nodeStatusDirty,
				hash:   make([]byte, common.HashLength),
			},
			Key:   key,
			Value: value,
		}
		return true, newnode, nil
	}
	return false, nil, nil
}

func (t *Trie) insertExtensionNode(n *ExtensionNode, key []byte, value []byte) (bool, noder, error) {
	matchlen := matchkeyLen(n.Key, key)
	if matchlen == len(n.Key) { // key match insert in nextNode
		var dirty bool
		dirty, n.NextNode, _ = t.insert(n.NextNode, key[matchlen:], value)
		if dirty {
			n.status = nodeStatusDirty
		}
		return dirty, n, nil
	}
	branchnode := &BranchNode{
		Node: Node{
			status: nodeStatusDirty,
			hash:   make([]byte, common.HashLength),
		},
	}

	if matchlen != len(n.Key)-1 {
		branchnode.Children[n.Key[matchlen]] = n
		n.Key = n.Key[matchlen+1:]
		n.status = nodeStatusDirty
	} else {
		branchnode.Children[n.Key[matchlen]] = n.NextNode
	}

	var err error
	_, branchnode.Children[key[matchlen]], err = t.insert(nil, key[matchlen+1:], value)
	if err != nil {
		return false, nil, err
	}
	if matchlen == 0 { // not match key value return branch node
		return true, branchnode, nil
	}

	return true, &ExtensionNode{ // have match key,return extension node
		Node: Node{
			status: nodeStatusDirty,
			hash:   make([]byte, common.HashLength),
		},
		Key:      key[:matchlen],
		NextNode: branchnode,
	}, nil
}

func (t *Trie) insertLeafNode(n *LeafNode, key []byte, value []byte) (bool, noder, error) {
	matchlen := matchkeyLen(n.Key, key)
	if matchlen == len(n.Key) { // key match, change the value of leaf node
		n.Value = value
		n.status = nodeStatusDirty
		return true, n, nil
	}
	branchnode := &BranchNode{
		Node: Node{
			status: nodeStatusDirty,
			hash:   make([]byte, common.HashLength),
		},
	}
	var err error
	branchnode.Children[n.Key[matchlen]] = n
	n.Key = n.Key[matchlen+1:]
	n.status = nodeStatusDirty

	_, branchnode.Children[key[matchlen]], err = t.insert(nil, key[matchlen+1:], value)
	if err != nil {
		return false, nil, err
	}
	if matchlen == 0 { // not match key value return branch node
		return true, branchnode, nil
	}

	return true, &ExtensionNode{ // have match key,return extension node
		Node: Node{
			status: nodeStatusDirty,
			hash:   make([]byte, common.HashLength),
		},
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
			if err == nil && match {
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
			if err == nil {
				n.Children[key[0]] = newnode
			}
			if !match {
				return false, n, nil
			}
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
					Node: Node{
						status: nodeStatusDirty,
						hash:   make([]byte, common.HashLength),
					},
					Key:   append([]byte{byte(pos)}, childnode.Key...),
					Value: childnode.Value,
				}
				return true, newnode, nil
			case *ExtensionNode:
				newnode := &ExtensionNode{
					Node: Node{
						status: nodeStatusDirty,
						hash:   make([]byte, common.HashLength),
					},
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
		match, newnode, err := t.delete(loadnode, key, descendant)
		if err != nil {
			return false, loadnode, err
		}
		return match, newnode, nil
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
	return t.decodeNode(hash, val)
}

// decodeNode decode node from buf byte
func (t *Trie) decodeNode(hash, value []byte) (noder, error) {
	if len(value) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	vals, _, err := rlp.SplitList(value)
	if err != nil {
		return nil, err
	}
	switch n, _ := rlp.CountValues(vals); n {
	case 1:
		return t.decodeBranchNode(hash, vals)
	case 2:
		return t.decodeLeafNode(hash, vals)
	case 3:
		return t.decodeExtensionNode(hash, vals)
	default:
		return nil, nil
	}
}

func (t *Trie) decodeLeafNode(hash, values []byte) (noder, error) {
	key, rest, err := rlp.SplitString(values)
	if err != nil {
		return nil, err
	}
	val, _, err := rlp.SplitString(rest)
	if err != nil {
		return nil, err
	}
	return &LeafNode{
		Node: Node{
			status: nodeStatusPersisted,
			hash:   hash,
		},
		Key:   key,
		Value: val,
	}, nil
}

func (t *Trie) decodeExtensionNode(hash, values []byte) (noder, error) {
	_, bufs, err := rlp.SplitString(values)
	key, rest, err := rlp.SplitString(bufs)
	if err != nil {
		return nil, err
	}
	val, _, err := rlp.SplitString(rest)
	if err != nil {
		return nil, err
	}
	return &ExtensionNode{
		Node: Node{
			status: nodeStatusPersisted,
			hash:   hash,
		},
		Key:      key,
		NextNode: append(hashNode{}, val...),
	}, nil
}

func (t *Trie) decodeBranchNode(hash, values []byte) (noder, error) {

	kind, elems, _, err := rlp.Split(values)
	if err != nil {
		return nil, err
	}
	itemcount, _ := rlp.CountValues(elems)
	if kind != rlp.List && itemcount != numBranchChildren {
		return nil, errNodeFormat
	}
	branchnode := &BranchNode{
		Node: Node{
			status: nodeStatusPersisted,
			hash:   hash,
		},
	}
	for i := 0; i < numBranchChildren; i++ {
		kind, val, rest, err := rlp.Split(elems)
		if err != nil {
			return nil, err
		}
		elems = rest
		if kind == rlp.String {
			length := len(val)
			if length == common.HashLength {
				branchnode.Children[i] = append(hashNode{}, val...)
			} else {
				branchnode.Children[i] = nil
			}
		}
	}
	return branchnode, nil
}

func (t *Trie) get(node noder, key []byte, pos int) (value []byte, newnode noder) {
	switch n := (node).(type) {
	case nil:
		return nil, nil
	case *ExtensionNode:
		if len(key)-pos < len(n.Key) || !bytes.Equal(n.Key, key[pos:pos+len(n.Key)]) {
			return nil, n
		}
		val, newnode := t.get(n.NextNode, key, pos+len(n.Key))
		n.NextNode = newnode
		return val, n
	case hashNode:
		child, err := t.loadNode(n)
		if err != nil {
			return nil, n
		}
		val, _ := t.get(child, key, pos)
		return val, child
	case *LeafNode:
		if len(key)-pos < len(n.Key) || !bytes.Equal(n.Key, key[pos:pos+len(n.Key)]) {
			// key not found in trie
			return nil, n
		}
		return n.Value, n
	case *BranchNode:
		val, newnode := t.get(n.Children[key[pos]], key, pos+1)
		n.Children[key[pos]] = newnode
		return val, n
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
