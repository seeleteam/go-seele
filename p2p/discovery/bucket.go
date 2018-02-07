/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import ()

const (
	bucketSize = 16 // Kademlia bucket size
)

type bucket struct {
	peers []*Node
}

func NewBuckets() *bucket {
	return &bucket{
		peers: make([]*Node, 0),
	}
}

// addNode add node to bucket, if bucket is full, will remove an old one
func (b *bucket) addNode(node *Node) {
	index := b.hasNode(node)

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

// hasNode check if the bucket already have this node, if so, return its index, otherwise, return -1
func (b *bucket) hasNode(node *Node) int {
	for index, n := range b.peers {
		if n.sha == node.sha {
			return index
		}
	}

	return -1
}
