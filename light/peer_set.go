/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"math/big"
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

func (p *peerSet) bestPeer(shard uint) *peer {
	var (
		bestPeer *peer
		bestTd   *big.Int
	)

	p.ForEach(shard, func(p *peer) bool {
		if _, td := p.Head(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			if !p.isSyncing() {
				bestPeer, bestTd = p, td
			}
		}

		return true
	})

	return bestPeer
}

func (p *peerSet) ForEach(shard uint, handle func(*peer) bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, v := range p.shardPeers[shard] {
		if !handle(v) {
			break
		}
	}
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

func (p *peerSet) choosePeers(shard uint, hash common.Hash) (choosePeers []*peer) {
	p.lock.Lock()
	defer p.lock.Unlock()

	mapLen := len(p.shardPeers[shard])
	peerL := make([]*peer, mapLen)

	idx := 0
	for _, v := range p.shardPeers[shard] {
		peerL[idx] = v
		idx++
	}

	common.Shuffle(peerL)
	cnt := 0
	for _, p := range peerL {
		if p.findIdxByHash(hash) >= 0 {
			cnt++
			choosePeers = append(choosePeers, p)
			if cnt >= 3 {
				return
			}
		}
	}

	return
}
