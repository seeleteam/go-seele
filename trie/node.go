/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package trie

const (
	// numBranchNodes number children in branch node
	numBranchNodes int = 17 // for 0-f branches + value node; reduce the height of tree for performance
)

// Noder interface for node
type noder interface {
	Hash() []byte
	IsDirty() bool // node is just created;value/childern is modified;it return true.
}

// Node is trie node struct
type Node struct {
	hash  []byte // hash of the node
	dirty bool   // is the node dirty,need to write to database
}

// ExtendNode is extend node struct
type ExtendNode struct {
	Node
	Key      []byte // for shared nibbles
	Nextnode noder  // for next node
}

// LeafNode is leaf node struct
type LeafNode struct {
	Node
	Key   []byte // for key-end
	Value []byte // the value of leafnode
}

// BranchNode is node for branch
type BranchNode struct {
	Node
	Children [numBranchNodes]noder
}

// hashNode is just used by nextnode of ExtendNode
// when it does not load real node from database
type hashNode []byte

// Hash return the hash of node
func (n hashNode) Hash() []byte {
	return n
}

// IsDirty is node dirty
func (n hashNode) IsDirty() bool {
	return false
}

// Hash return the hash of node
func (n Node) Hash() []byte {
	return n.hash
}

// IsDirty is node dirty
func (n Node) IsDirty() bool {
	return n.dirty
}
