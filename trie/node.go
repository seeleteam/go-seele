/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package trie

const (
	// numBranchChildren number children in branch node
	numBranchChildren int = 17 // for 0-f branches + value node; reduce the height of tree for performance

	nodeStatusDirty     nodeStatus = iota // node is newly created or modified, but not update the node hash
	nodeStatusUpdated                     // node hash updated
	nodeStatusPersisted                   // node persisted in DB, in which case the node hash also updated
)

type nodeStatus byte

// Noder interface for node
type noder interface {
	Hash() []byte
	Status() nodeStatus
}

// Node is trie node struct
type Node struct {
	hash   []byte     // hash of the node
	status nodeStatus // status of the node
}

// ExtensionNode is extension node struct
type ExtensionNode struct {
	Node
	Key      []byte // for shared nibbles
	NextNode noder  // for next node
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
	Children [numBranchChildren]noder
}

// hashNode is just used by NextNode of ExtensionNode
// when it does not load real node from database
type hashNode []byte

// Hash return the hash of node
func (n hashNode) Hash() []byte {
	return n
}

// Status return the status of node
func (n hashNode) Status() nodeStatus {
	return nodeStatusPersisted
}

// Hash return the hash of node
func (n Node) Hash() []byte {
	return n.hash
}

// Status return the status of node
func (n Node) Status() nodeStatus {
	return n.status
}
