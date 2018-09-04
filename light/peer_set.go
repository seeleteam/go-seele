/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"sync"

	"github.com/seeleteam/go-seele/common"
)

type peerSet struct {
	peerMap    map[common.Address]*peer
	shardPeers [1 + common.ShardCount]map[common.Address]*peer
	lock       sync.RWMutex
}

func newPeerSet() *peerSet {
	ps := &peerSet{
		peerMap: make(map[common.Address]*peer),
		lock:    sync.RWMutex{},
	}

	for i := 0; i < 1+common.ShardCount; i++ {
		ps.shardPeers[i] = make(map[common.Address]*peer)
	}

	return ps
}

func (p *peerSet) Remove(peerID common.Address) {
	p.lock.Lock()
	defer p.lock.Unlock()

	result := p.peerMap[peerID]
	if result != nil {
		delete(p.peerMap, peerID)
		delete(p.shardPeers[result.Node.Shard], peerID)
	}
}

func (p *peerSet) Add(pe *peer) {
	p.lock.Lock()
	defer p.lock.Unlock()

	peerID := pe.peerID
	result := p.peerMap[peerID]
	if result != nil {
		delete(p.peerMap, peerID)
		delete(p.shardPeers[result.Node.Shard], peerID)
	}

	p.peerMap[peerID] = pe
	p.shardPeers[pe.Node.Shard][peerID] = pe
}

func (p *peerSet) Find(address common.Address) *peer {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.peerMap[address]
}

func (p *peerSet) choosePeers() []*peer {
	p.lock.Lock()
	defer p.lock.Unlock()
	// todo choose peers randomly
	return nil
}
