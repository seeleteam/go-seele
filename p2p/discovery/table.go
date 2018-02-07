/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"
	"sort"

	"github.com/seeleteam/go-seele/common"
)

const (
	alpha      = 3  // Kademlia concurrency factor
	responseNodeNumber = 5 //TODO with this number for test
	hashBits   = len(common.Hash{}) * 8
	nBuckets   = hashBits + 1 // Number of buckets
)

type Table struct {
	buckets  [nBuckets]*bucket
	count    int   //total number of nodes
	selfNode *Node //info of local node
}

func NewTable(id NodeID, addr *net.UDPAddr) *Table {
	selfNode := NewNodeWithAddr(id, addr)

	table := &Table{
		count:    0,
		selfNode: selfNode,
	}

	for i := 0; i < nBuckets; i++ {
		table.buckets[i] = NewBuckets()
	}

	return table
}

func (t *Table) addNode(node *Node) {
	dis := logdist(node.sha, t.selfNode.sha)

	t.buckets[dis].addNode(node)
}

func (t *Table) updateNode(node *Node)  {
	t.addNode(node)
}

func (t *Table) findNodeResponse(target common.Hash) []*Node {
	nodes := t.findMinDisNodes(target, responseNodeNumber)

	minDis := []*Node{}
	for _, e := range nodes {
		if distcmp(target, t.selfNode.sha, e.sha) > 0 {
			minDis = append(minDis, e)
		}
	}

	return minDis
}

func (t *Table) deleteNode(target common.Hash)  {
	dis := logdist(t.selfNode.sha, target)

	index := 0
	for i, n := range t.buckets[dis].peers {
		if n.sha == target {
			index = i
			break
		}
	}

	t.buckets[dis].peers = append(t.buckets[dis].peers[:index], t.buckets[dis].peers[index+1:]...)
}

func (t *Table) findNodeForRequest(target common.Hash) []*Node {
	return t.findMinDisNodes(target, alpha)
}

func (t *Table) findMinDisNodes(target common.Hash, number int) []*Node  {
	result := nodesByDistance{
		target: target,
		maxElems: number,
		entries: make([]*Node, 0),
	}

	for _, b := range t.buckets {
		for _, n := range b.peers {
			result.push(n)
		}
	}

	return result.entries
}


// nodesByDistance is a list of nodes, ordered by
// distance to target.
type nodesByDistance struct {
	entries []*Node
	target  common.Hash
	maxElems int
}

// push adds the given node to the list, keeping the total size below maxElems.
func (h *nodesByDistance) push(n *Node) {
	ix := sort.Search(len(h.entries), func(i int) bool {
		return distcmp(h.target, h.entries[i].sha, n.sha) > 0
	})
	if len(h.entries) < h.maxElems {
		h.entries = append(h.entries, n)
	}
	if ix == len(h.entries) {
		// farther away than all nodes we already have.
		// if there was room for it, the node is now the last element.
	} else {
		// slide existing entries down to make room
		// this will overwrite the entry we just appended.
		copy(h.entries[ix+1:], h.entries[ix:])
		h.entries[ix] = n
	}
}
