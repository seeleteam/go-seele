/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/crypto"
)

const (
	bucketSize = 16 // Kademlia bucket size
)

type bucket struct {
	peers []*Node
	lock  sync.RWMutex //used for peers change
}

func newBuckets() *bucket {
	return &bucket{
		peers: make([]*Node, 0),
		lock:  sync.RWMutex{},
	}
}

// addNode add node to bucket, if bucket is full, will remove an old one
func (b *bucket) addNode(node *Node) {
	index := b.findNode(node)

	if index != -1 {
		// do nothing for now
		// TODO lru
	} else {
		b.lock.Lock()
		defer b.lock.Unlock()

		log.Info("add node: %s", hexutil.BytesToHex(node.ID.Bytes()))
		if len(b.peers) < bucketSize {
			b.peers = append(b.peers, node)
		} else {
			copy(b.peers[:], b.peers[1:])
			b.peers[len(b.peers)-1] = node
		}
	}
}

// findNode check if the bucket already have this node, if so, return its index, otherwise, return -1
func (b *bucket) findNode(node *Node) int {
	b.lock.RLock()
	defer b.lock.RUnlock()
	for index, n := range b.peers {
		if n.ID == node.ID {
			return index
		}
	}

	return -1
}

func (b *bucket) deleteNode(target common.Hash) {
	b.lock.Lock()
	defer b.lock.Unlock()

	index := -1
	for i, n := range b.peers {
		sha := crypto.HashBytes(n.ID.Bytes())
		if sha == target {
			index = i
			break
		}
	}

	if index == -1 {
		log.Error("Failed to find the node to delete\n")
		return
	}

	log.Info("delete node: %s", hexutil.BytesToHex(b.peers[index].ID.Bytes()))

	b.peers = append(b.peers[:index], b.peers[index+1:]...)
}

func (b *bucket) size() int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return len(b.peers)
}

// printNodeList only use for debug test
func (b *bucket) printNodeList() {
	log.Debug("bucket size %d", len(b.peers))

	for _, n := range b.peers {
		log.Debug("%s", hexutil.BytesToHex(n.ID.Bytes()))
	}
}
