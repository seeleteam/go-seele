/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/log"
	"sync"
)

const (
	bucketSize = 16 // Kademlia bucket size
)

type bucket struct {
	peers []*Node

	lock sync.Mutex	//used for peers change
}

func NewBuckets() *bucket {
	return &bucket{
		peers: make([]*Node, 0),
		lock: sync.Mutex{},
	}
}

// addNode add node to bucket, if bucket is full, will remove an old one
func (b *bucket) addNode(node *Node) {
	index := b.hasNode(node)

	if index != -1 {
		// do nothing for now
		// TODO lru
	} else {
		b.lock.Lock()
		defer b.lock.Unlock()

		if len(b.peers) < bucketSize {
			b.peers = append(b.peers, node)
		} else {
			copy(b.peers[:], b.peers[1:])
			b.peers[len(b.peers)-1] = node
		}
	}

	b.printNodeList()
}

// hasNode check if the bucket already have this node, if so, return its index, otherwise, return -1
func (b *bucket) hasNode(node *Node) int {
	b.lock.Lock()
	defer b.lock.Unlock()
	for index, n := range b.peers {
		if n.ID == node.ID {
			return index
		}
	}

	return -1
}

func (b *bucket) deleteNode(target *common.Hash) {
	b.lock.Lock()
	defer b.lock.Unlock()

	index := -1
	for i, n := range b.peers {
		sha := n.ID.ToSha()
		if *sha == *target {
			index = i
			break
		}
	}

	if index == -1 {
		log.Panic("don't find the node to delete\n")
		b.printNodeList()
		return
	}

	b.peers = append(b.peers[:index], b.peers[index+1:]...)

	b.printNodeList()
}

func (b *bucket) size() int {
	b.lock.Lock()
	defer b.lock.Unlock()

	return len(b.peers)
}

// printNodeList only use for debug test
func (b *bucket) printNodeList() {
	log.Debug("bucket size %d", len(b.peers))

	for _, n := range b.peers {
		log.Debug("%s", hexutil.BytesToHex(n.ID.Bytes()))
	}
}