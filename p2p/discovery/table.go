/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"
	"sort"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
)

const (
	// Kademlia concurrency factor
	alpha = 3
	// TODO with this number for test
	responseNodeNumber = 10
	hashBits           = len(common.Hash{}) * 8
	// Height of buckets
	nBuckets = hashBits + 1
	// other shard minimal node number for start
	shardTargeNodeNumber = 1
	// UndefinedShardNumber indicates the shard number is undefined
	UndefinedShardNumber = 0
)

// Table used to save peers information
type Table struct {
	buckets [nBuckets]*bucket
	// 0 represents undefined shard number node.
	shardBuckets [common.ShardCount + 1]*bucket
	// info of local node
	selfNode *Node

	log *log.SeeleLog
}

func newTable(id common.Address, addr *net.UDPAddr, shard uint, log *log.SeeleLog) *Table {
	selfNode := NewNodeWithAddr(id, addr, shard)

	table := &Table{
		selfNode: selfNode,
		log:      log,
	}

	for i := 0; i < nBuckets; i++ {
		table.buckets[i] = newBuckets(log)
	}

	for i := 0; i < common.ShardCount+1; i++ {
		table.shardBuckets[i] = newBuckets(log)
	}

	return table
}

type nodesConnectedByPeers struct {
	target  common.Hash
	entries []*Node
}

// findConnectedNodes return the responseNodeNumber connected nodes
// @repc_nodes
func (t *Table) findConnectedNodes(target common.Hash) []*Node {
	result := nodesConnectedByPeers{
		target:  target,
		entries: make([]*Node, 0),
	}

	for _, b := range t.buckets {
		for _, n := range b.peers {
			result.entries = append(result.entries, n)
		}
	}

	return result.entries
}

func (t *Table) addNode(node *Node) bool {
	if isShardValid(node.Shard) {
		if node.Shard != t.selfNode.Shard {
			t.shardBuckets[node.Shard].addNode(node)

		} else {
			dis := logDist(t.selfNode.getSha(), node.getSha())

			t.buckets[dis].addNode(node)
		}
		// the node is in the buckets
		return true
	} else {
		t.log.Debug("get invalid shard, shard count is %d, getting shard number is %d", common.ShardCount, node.Shard)
	}
	return false
}

// getPeersCount obtain all peers count
func (t *Table) count() int {
	count := 0
	for _, v := range t.buckets {
		count += len(v.peers)
	}

	for _, v := range t.shardBuckets {
		count += len(v.peers)
	}

	return count
}

func (t *Table) updateNode(node *Node) {
	t.addNode(node)
}

// findNodeWithTarget find nodes that distance of target is less than measure with target
func (t *Table) findNodeWithTarget(target common.Hash) []*Node {
	nodes := t.findMinDisNodes(target, responseNodeNumber)

	minDis := []*Node{}
	for _, e := range nodes {
		if distCmp(target, t.selfNode.getSha(), e.getSha()) > 0 {
			minDis = append(minDis, e)
		}
	}

	return minDis
}

func (t *Table) deleteNode(n *Node) {
	sha := n.getSha()
	if isShardValid(n.Shard) {
		if n.Shard != t.selfNode.Shard {
			t.shardBuckets[n.Shard].deleteNode(sha)
		} else {
			dis := logDist(t.selfNode.getSha(), sha)
			t.buckets[dis].deleteNode(sha)
		}
	}
}

// findNodeForRequest calls when start find node, find the initialize nodes
func (t *Table) findNodeForRequest(target common.Hash) []*Node {
	return t.findMinDisNodes(target, alpha)
}

func (t *Table) findMinDisNodes(target common.Hash, number int) []*Node {
	result := nodesByDistance{
		target:   target,
		maxElems: number,
		entries:  make([]*Node, 0),
	}

	for _, b := range t.buckets {
		for _, n := range b.peers {
			result.push(n)
		}
	}

	return result.entries
}

func (t *Table) GetRandNodes(number int) []*Node {
	// TODO get nodes randomly
	nodes := make([]*Node, 0)
	count := 0
	for i := 0; i < nBuckets; i++ {
		b := t.buckets[i]
		if b.size() > 0 {
			bnodes := b.getRandNodes(number)

			for j := 0; j < len(bnodes); j++ {
				nodes = append(nodes, bnodes[j])
				count++

				if count == number {
					return nodes
				}
			}
		}
	}

	return nodes
}

// nodesByDistance is a list of nodes, ordered by
// distance to to.
type nodesByDistance struct {
	entries  []*Node
	target   common.Hash
	maxElems int
}

// push adds the given node to the list, keeping the total size below maxElems.
func (h *nodesByDistance) push(n *Node) {
	ix := sort.Search(len(h.entries), func(i int) bool {
		return distCmp(h.target, h.entries[i].getSha(), n.getSha()) > 0
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
