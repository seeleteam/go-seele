/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"

	"github.com/seeleteam/go-seele/common"
)

const (
	alpha      = 3  // Kademlia concurrency factor
	bucketSize = 16 // Kademlia bucket size
	hashBits   = len(common.Hash{}) * 8
	nBuckets   = hashBits + 1 // Number of buckets
)

type Table struct {
	buckets  [nBuckets]*bucket
	count    int   //total number of nodes
	selfNode *Node //info of local node
}

type bucket struct {
	peers []*Node
}

func NewTable(id NodeID, addr *net.UDPAddr) *Table {
	selfNode := NewNode(id, addr)

	table := &Table{
		count:    0,
		selfNode: selfNode,
	}

	return table
}

func (t *Table) AddNode(node *Node) {
	dis := logdist(node.sha, t.selfNode.sha)

	t.buckets[dis].AddNode(node)
}

// AddNode add node to bucket, if bucket is full, will remove an old one
func (b *bucket) AddNode(node *Node) {
	index := b.HasNode(node)

	if index != -1 {
		// do nothing for now
		// TODO lru
	} else {
		if len(b.peers) < bucketSize {
			b.peers = append(b.peers, node)
		} else {
			copy(b.peers[:], b.peers[1:])
			b.peers[len(b.peers)-1] = node
		}
	}
}

// HasNode check if the bucket already have this node, if so, return its index, otherwise, return -1
func (b *bucket) HasNode(node *Node) int {
	for index, n := range b.peers {
		if n.sha == node.sha {
			return index
		}
	}

	return -1
}
