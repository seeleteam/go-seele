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
	Hash() []byte                // return the node hash
	Status() nodeStatus          // return the node status
	SetHash(hash []byte)         // update the node hash
	SetStatus(status nodeStatus) // update the node status
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

// SetHash do nothing
func (n hashNode) SetHash(hash []byte) {
	panic("hashnode do not support to change hash")
}

// SetStatus do nothing
func (n hashNode) SetStatus(status nodeStatus) {
	panic("hashnode do not support to change status")
}

// Hash return the hash of node
func (n *Node) Hash() []byte {
	return n.hash
}

// Status return the status of node
func (n *Node) Status() nodeStatus {
	return n.status
}

// SetHash set the node hash
func (n *Node) SetHash(hash []byte) {
	copy(n.hash, hash)
}

// SetStatus set the node status
func (n *Node) SetStatus(status nodeStatus) {
	n.status = status
}
