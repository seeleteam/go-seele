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
	log2 "github.com/seeleteam/go-seele/log"
)

const (
	bucketSize = 16 // Kademlia bucket size
)

type bucket struct {
	peers []*Node
	lock  sync.RWMutex //used for peers change

	log *log2.SeeleLog
}

func newBuckets(log *log2.SeeleLog) *bucket {
	return &bucket{
		peers: make([]*Node, 0),
		lock:  sync.RWMutex{},
		log:   log,
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
		b.log.Error("Failed to find the node to delete")
		return
	}

	b.log.Info("delete node: %s", hexutil.BytesToHex(b.peers[index].ID.Bytes()))

	b.peers = append(b.peers[:index], b.peers[index+1:]...)
}

func (b *bucket) getRandNodes(number int) []*Node {
	b.lock.RLock()
	defer b.lock.RUnlock()

	var result []*Node
	if len(b.peers) > number {
		result = make([]*Node, number)
		// @TODO use random selection
		for i := 0; i < number; i++ {
			result[i] = &Node{}
			*result[i] = *b.peers[i]
		}
	} else {
		result = make([]*Node, len(b.peers))
		for i := 0; i < len(result); i++ {
			result[i] = &Node{}
			*(result[i]) = *(b.peers[i])
		}
	}

	return result
}

func (b *bucket) get(index int) *Node {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if index < len(b.peers) {
		return b.peers[index]
	}

	return nil
}

func (b *bucket) size() int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return len(b.peers)
}

// printNodeList only use for debug test
func (b *bucket) printNodeList() {
	b.log.Debug("bucket size %d", len(b.peers))

	for _, n := range b.peers {
		b.log.Debug("%s", hexutil.BytesToHex(n.ID.Bytes()))
	}
}
