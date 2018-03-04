/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package merkle

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/seeleteam/go-seele/common"
)

var (
	errNoContent = errors.New("Error: cannot construct tree with no content.")
)

// Content represents the data that is stored and verified by the tree. A type that
// implements this interface can be used as an item in the tree.
type Content interface {
	CalculateHash() common.Hash
	Equals(other Content) bool
}

// MerkleTree is the container for the tree. It holds a pointer to the root of the tree,
// a list of pointers to the leaf nodes, and the merkle root.
// Note, it is not thread safe
type MerkleTree struct {
	Root       *node
	merkleRoot common.Hash
	Leafs      []*node
}

// node represents a node, root, or leaf in the tree. It stores pointers to its immediate
// relationships, a hash, the content stored if it is a leaf, and other metadata.
type node struct {
	Parent  *node
	Left    *node
	Right   *node
	dup     bool // indicates that whether this is a duplicate node in the rightmost leaf of the tree.
	Hash    common.Hash
	Content Content
}

func (n *node) isLeaf() bool {
	return n.Content != nil
}

// calculateHashRecursively walks down the tree until hitting a leaf, calculating the hash at each level
// and returning the resulting hash of node n.
func (n *node) calculateHashRecursively() common.Hash {
	if n.isLeaf() {
		return n.Content.CalculateHash()
	}
	return common.HashBytes(append(n.Left.calculateHashRecursively().Bytes(), n.Right.calculateHashRecursively().Bytes()...))
}

// calculateHash is a helper function that calculates the hash of the node.
func (n *node) calculateHash() common.Hash {
	if n.isLeaf() {
		return n.Content.CalculateHash()
	}
	return common.HashBytes(append(n.Left.Hash.Bytes(), n.Right.Hash.Bytes()...))
}

// NewTree creates a new Merkle Tree using the content cs.
func NewTree(contents []Content) (*MerkleTree, error) {
	root, leafs, err := buildWithContent(contents)
	if err != nil {
		return nil, err
	}
	t := &MerkleTree{
		Root:       root,
		merkleRoot: root.Hash,
		Leafs:      leafs,
	}
	return t, nil
}

// buildWithContent is a helper function that for a given set of Contents, to generates a
// corresponding tree and returns the root node, a list of leaf nodes, and a possible error.
// Returns an error if contents contains no Contents.
func buildWithContent(contents []Content) (*node, []*node, error) {
	if len(contents) == 0 {
		return nil, nil, errNoContent
	}
	var leafs []*node
	for _, c := range contents {
		leafs = append(leafs, &node{
			Hash:    c.CalculateHash(),
			Content: c,
		})
	}
	if len(leafs)%2 == 1 {
		duplicate := &node{
			Hash:    leafs[len(leafs)-1].Hash,
			Content: leafs[len(leafs)-1].Content,
			dup:     true,
		}
		leafs = append(leafs, duplicate)
	}
	root := buildIntermediate(leafs)
	return root, leafs, nil
}

// buildIntermediate is a helper function that for a given list of leaf nodes, constructs
// the intermediate and root levels of the tree. Returns the resulting root node of the tree.
func buildIntermediate(nodeList []*node) *node {
	var nodes []*node
	for i := 0; i < len(nodeList); i += 2 {
		var left, right int = i, i + 1
		if i+1 == len(nodeList) {
			right = i
		}
		chash := append(nodeList[left].Hash.Bytes(), nodeList[right].Hash.Bytes()...)
		n := &node{
			Left:  nodeList[left],
			Right: nodeList[right],
			Hash:  common.HashBytes(chash),
		}
		nodes = append(nodes, n)
		nodeList[left].Parent = n
		nodeList[right].Parent = n
		if len(nodeList) == 2 {
			return n
		}
	}
	return buildIntermediate(nodes)
}

// MerkleRoot returns the unverified Merkle Root (hash of the root node) of the tree.
func (m *MerkleTree) MerkleRoot() common.Hash {
	return m.merkleRoot
}

// RebuildTree is a helper function that will rebuild the tree reusing only the content that
// it holds in the leaves.
func (m *MerkleTree) RebuildTree() error {
	var cs []Content
	for _, c := range m.Leafs {
		cs = append(cs, c.Content)
	}
	root, leafs, err := buildWithContent(cs)
	if err != nil {
		return err
	}
	m.Root = root
	m.Leafs = leafs
	m.merkleRoot = root.Hash
	return nil
}

// RebuildTreeWith replaces the content of the tree and does a complete rebuild; while the root of
// the tree will be replaced the MerkleTree completely survives this operation. Returns an error if the
// list of content cs contains no entries.
func (m *MerkleTree) RebuildTreeWith(cs []Content) error {
	root, leafs, err := buildWithContent(cs)
	if err != nil {
		return err
	}
	m.Root = root
	m.Leafs = leafs
	m.merkleRoot = root.Hash
	return nil
}

// VerifyTree validates the hashes at each level of the tree and returns true if the
// resulting hash at the root of the tree matches the resulting root hash; otherwise, returns false.
func (m *MerkleTree) VerifyTree() bool {
	calculatedMerkleRoot := m.Root.calculateHashRecursively()

	return bytes.Compare(m.merkleRoot.Bytes(), calculatedMerkleRoot.Bytes()) == 0
}

// VerifyContent indicates whether a given content is in the tree and the hashes are valid for that content.
// Returns true if the expected Merkle Root is equivalent to the Merkle root calculated on the critical path
// for a given content. Returns true if valid and false otherwise.
func (m *MerkleTree) VerifyContent(expectedMerkleRoot []byte, content Content) bool {
	if bytes.Compare(m.merkleRoot.Bytes(), expectedMerkleRoot) != 0 {
		return false
	}

	for _, l := range m.Leafs {
		if l.Content.Equals(content) {
			currentParent := l.Parent
			for currentParent != nil {
				buff := append(currentParent.Left.calculateHash().Bytes(), currentParent.Right.calculateHash().Bytes()...)
				if bytes.Compare(common.HashBytes(buff).Bytes(), currentParent.Hash.Bytes()) != 0 {
					return false
				}
				currentParent = currentParent.Parent
			}
			return true
		}
	}
	return false
}

// String returns a string representation of the tree. Only leaf nodes are included
// in the output.
func (m *MerkleTree) String() string {
	s := ""
	for _, l := range m.Leafs {
		s += fmt.Sprintln(l)
	}
	return s
}
