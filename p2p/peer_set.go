/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"sync"

	"github.com/seeleteam/go-seele/common"
)

// peerSet is thread safe collection
type peerSet struct {
	peerMap      map[common.Address]*Peer
	shardPeerMap map[uint]map[common.Address]*Peer
	lock         sync.RWMutex
}

func NewPeerSet() *peerSet {
	peers := make(map[uint]map[common.Address]*Peer)
	for i := 1; i < common.ShardCount+1; i++ {
		peers[uint(i)] = make(map[common.Address]*Peer)
	}

	return &peerSet{
		peerMap:      make(map[common.Address]*Peer),
		shardPeerMap: peers,
		lock:         sync.RWMutex{},
	}
}

func (set *peerSet) add(p *Peer) {
	set.lock.Lock()
	defer set.lock.Unlock()

	set.shardPeerMap[p.getShardNumber()][p.Node.ID] = p
	set.peerMap[p.Node.ID] = p
}

func (set *peerSet) count() int {
	set.lock.RLock()
	defer set.lock.RUnlock()

	return len(set.peerMap)
}

func (set *peerSet) find(addr common.Address) *Peer {
	set.lock.RLock()
	defer set.lock.RUnlock()

	return set.peerMap[addr]
}

func (set *peerSet) delete(p *Peer) {
	set.lock.Lock()
	defer set.lock.Unlock()

	delete(set.peerMap, p.Node.ID)
	delete(set.shardPeerMap[p.getShardNumber()], p.Node.ID)
}

func (set *peerSet) foreach(call func(p *Peer)) {
	set.lock.RLock()
	defer set.lock.RUnlock()

	for _, p := range set.peerMap {
		call(p)
	}
}
