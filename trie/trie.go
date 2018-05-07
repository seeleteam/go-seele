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

// GetDB returns database interface of trie 
func (t *Trie) GetDB() database.Database{
	return t.db
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
		match, newnode, err := t.delete(t.root, key)
		if err == nil && match {
			t.root = newnode
		}
		return match
	}
	return false
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

// Commit commit the dirty node to database
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
	if !node.IsDirty() {
		return node.Hash()
	}
	switch n := node.(type) {
	case *LeafNode:
		buf.Reset()
		rlp.Encode(buf, []interface{}{
			n.Key,
			n.Value,
		})
		sha.Reset()
		sha.Write(buf.Bytes())
		hash := sha.Sum(nil)
		if batch != nil {
			batch.Put(append(t.dbprefix, hash...), buf.Bytes())
			n.dirty = false
		}
		copy(n.hash, hash)
		return n.hash
	case *ExtendNode:
		nexthash := t.hash(n.Nextnode, buf, sha, batch)
		buf.Reset()
		rlp.Encode(buf, []interface{}{
			true, //add it to diff with extend node;modify later using compact func?
			n.Key,
			nexthash,
		})
		sha.Reset()
		sha.Write(buf.Bytes())
		hash := sha.Sum(nil)
		if batch != nil {
			batch.Put(append(t.dbprefix, hash...), buf.Bytes())
			n.dirty = false
		}
		copy(n.hash, hash)
		return n.hash
	case *BranchNode:
		var children [numBranchNodes][]byte
		for i, child := range n.Children {
			children[i] = t.hash(child, buf, sha, batch)
		}
		buf.Reset()
		rlp.Encode(buf, []interface{}{
			children,
		})

		sha.Reset()
		sha.Write(buf.Bytes())
		hash := sha.Sum(nil)
		if batch != nil {
			batch.Put(append(t.dbprefix, hash...), buf.Bytes())
			n.dirty = false
		}
		copy(n.hash, hash)
		return n.hash
	case hashNode:
		return n.Hash()
	default:
		panic(fmt.Sprintf("invalid node: %v", node))
	}
}

// return true if insert succeed,it also mean node is dirty,should recalc hash
func (t *Trie) insert(node noder, key []byte, value []byte) (bool, noder, error) {
	switch n := node.(type) {
	case *ExtendNode:
		return t.insertExtentNode(n, key, value)
	case *LeafNode:
		return t.insertLeafNode(n, key, value)
	case *BranchNode:
		_, child, err := t.insert(n.Children[key[0]], key[1:], value)
		if err != nil {
			return false, nil, err
		}
		n.Children[key[0]] = child
		n.dirty = true
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
				dirty: true,
				hash:  make([]byte, common.HashLength),
			},
			Key:   key,
			Value: value,
		}
		return true, newnode, nil
	}
	return false, nil, nil
}

func (t *Trie) insertExtentNode(n *ExtendNode, key []byte, value []byte) (bool, noder, error) {
	matchlen := matchkeyLen(n.Key, key)
	if matchlen == len(n.Key) { // key match insert in nextnode
		var dirty bool
		dirty, n.Nextnode, _ = t.insert(n.Nextnode, key[matchlen:], value)
		if dirty {
			n.dirty = true
		}
		return n.dirty, n, nil
	}
	branchnode := &BranchNode{
		Node: Node{
			dirty: true,
			hash:  make([]byte, common.HashLength),
		},
	}

	if matchlen != len(n.Key)-1 {
		branchnode.Children[n.Key[matchlen]] = n
		n.Key = n.Key[matchlen+1:]
		n.dirty = true
	} else {
		branchnode.Children[n.Key[matchlen]] = n.Nextnode
	}

	var err error
	_, branchnode.Children[key[matchlen]], err = t.insert(nil, key[matchlen+1:], value)
	if err != nil {
		return false, nil, err
	}
	if matchlen == 0 { // not match key value return branch node
		return true, branchnode, nil
	}

	return true, &ExtendNode{ // have match key,return extend node
		Node: Node{
			dirty: true,
			hash:  make([]byte, common.HashLength),
		},
		Key:      key[:matchlen],
		Nextnode: branchnode,
	}, nil
}

func (t *Trie) insertLeafNode(n *LeafNode, key []byte, value []byte) (bool, noder, error) {
	matchlen := matchkeyLen(n.Key, key)
	if matchlen == len(n.Key) { // key match, change the value of leaf node
		n.Value = value
		n.dirty = true
		return true, n, nil
	}
	branchnode := &BranchNode{
		Node: Node{
			dirty: true,
			hash:  make([]byte, common.HashLength),
		},
	}
	var err error
	branchnode.Children[n.Key[matchlen]] = n
	n.Key = n.Key[matchlen+1:]
	n.dirty = true

	_, branchnode.Children[key[matchlen]], err = t.insert(nil, key[matchlen+1:], value)
	if err != nil {
		return false, nil, err
	}
	if matchlen == 0 { // not match key value return branch node
		return true, branchnode, nil
	}

	return true, &ExtendNode{ // have match key,return extend node
		Node: Node{
			dirty: true,
			hash:  make([]byte, common.HashLength),
		},
		Key:      key[:matchlen],
		Nextnode: branchnode,
	}, nil
}

func (t *Trie) delete(node noder, key []byte) (bool, noder, error) {
	switch n := node.(type) {
	case *LeafNode:
		matchlen := matchkeyLen(key, n.Key)
		if matchlen == len(n.Key) {
			return true, nil, nil
		}
		return false, n, nil
	case *ExtendNode:
		matchlen := matchkeyLen(key, n.Key)
		if matchlen == len(n.Key) {
			match, newnode, err := t.delete(n.Nextnode, key[matchlen:])
			if err == nil && match {
				n.dirty = true
				n.Nextnode = newnode
				if newnode == nil {
					return true, nil, nil
				}
				return true, n, nil
			}
		}
		return false, n, nil
	case *BranchNode:
		match, newnode, err := t.delete(n.Children[key[0]], key[1:])
		if err == nil {
			n.Children[key[0]] = newnode
		}
		if match {
			n.dirty = true
		}
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
					return match, nil, err
				}
			}
			switch childnode := childnode.(type) {
			case *LeafNode:
				newnode := &LeafNode{
					Node: Node{
						dirty: true,
						hash:  make([]byte, common.HashLength),
					},
					Key:   append([]byte{byte(pos)}, childnode.Key...),
					Value: childnode.Value,
				}
				return true, newnode, nil
			case *ExtendNode:
				newnode := &ExtendNode{
					Node: Node{
						dirty: true,
						hash:  make([]byte, common.HashLength),
					},
					Key:      append([]byte{byte(pos)}, childnode.Key...),
					Nextnode: childnode.Nextnode,
				}
				return true, newnode, nil
			}
		}
		return match, n, nil
	case hashNode:
		loadnode, err := t.loadNode(n)
		if err != nil {
			return false, nil, err
		}
		match, newnode, err := t.delete(loadnode, key)
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
		return t.decodeExtendNode(hash, vals)
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
			dirty: false,
			hash:  hash,
		},
		Key:   key,
		Value: val,
	}, nil
}

func (t *Trie) decodeExtendNode(hash, values []byte) (noder, error) {
	_, bufs, err := rlp.SplitString(values)
	key, rest, err := rlp.SplitString(bufs)
	if err != nil {
		return nil, err
	}
	val, _, err := rlp.SplitString(rest)
	if err != nil {
		return nil, err
	}
	return &ExtendNode{
		Node: Node{
			dirty: false,
			hash:  hash,
		},
		Key:      key,
		Nextnode: append(hashNode{}, val...),
	}, nil
}

func (t *Trie) decodeBranchNode(hash, values []byte) (noder, error) {

	kind, elems, _, err := rlp.Split(values)
	if err != nil {
		return nil, err
	}
	itemcount, _ := rlp.CountValues(elems)
	if kind != rlp.List && itemcount != numBranchNodes {
		return nil, errNodeFormat
	}
	branchnode := &BranchNode{
		Node: Node{
			dirty: false,
			hash:  hash,
		},
	}
	for i := 0; i < numBranchNodes; i++ {
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
	case *ExtendNode:
		if len(key)-pos < len(n.Key) || !bytes.Equal(n.Key, key[pos:pos+len(n.Key)]) {
			return nil, n
		}
		val, newnode := t.get(n.Nextnode, key, pos+len(n.Key))
		n.Nextnode = newnode
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
		nibbles[i*2] = b / byte(numBranchNodes-1)   // now is b / 16
		nibbles[i*2+1] = b % byte(numBranchNodes-1) // now is b% 16
	}
	nibbles[l-1] = byte(numBranchNodes - 1) // term key is 16
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
