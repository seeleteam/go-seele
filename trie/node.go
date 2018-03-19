package trie

const (
	// NumberChildren number children in branch node
	NumberChildren int = 17
	// LengthOfNodeHash length of node hash
	LengthOfNodeHash int = 32
)

// Noder interface for node
type Noder interface {
	Hash() []byte
	IsDirty() bool
}

// Node is trie node struct
type Node struct {
	hash  []byte // hash of the node
	dirty bool   // is the node dirty,need to write to database
}

// ExtendNode is extend node struct.for root,extend node
type ExtendNode struct {
	Node
	Key      []byte // for shared nibbles or key-end
	Nextnode Noder
}

// LeafNode is leaf node struct
type LeafNode struct {
	Node
	Key   []byte // for shared nibbles or key-end
	Value []byte // the value of leafnode
}

// BranchNode is node for branch
type BranchNode struct {
	Node
	Children [NumberChildren]Noder
}

// hashNode is just used by nextnode of ExtendNode
// when it does not load real node from datbase
type hashNode []byte

// Hash return the hash of node
func (n hashNode) Hash() []byte { return n }

// IsDirty is node dirty
func (n hashNode) IsDirty() bool { return false }

// Hash return the hash of node
func (n Node) Hash() []byte {
	return n.hash
}

// IsDirty is node dirty
func (n Node) IsDirty() bool {
	return n.dirty
}
