/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"math/big"
	"math/rand"
	"sync"

	"github.com/seeleteam/go-seele/common"
)

type peerFilter struct {
	blockHash common.Hash
}

type peerSet struct {
	peerMap map[common.Address]*peer
	lock    sync.RWMutex
}

func newPeerSet() *peerSet {
	ps := &peerSet{
		peerMap: make(map[common.Address]*peer),
		lock:    sync.RWMutex{},
	}

	return ps
}

func (p *peerSet) getPeers() map[common.Address]*peer {
	p.lock.RLock()
	defer p.lock.RUnlock()

	value := make(map[common.Address]*peer)

	for key, v := range p.peerMap {
		value[key] = v
	}

	return value
}

func (p *peerSet) bestPeer() *peer {
	var (
		bestPeer *peer
		bestTd   *big.Int
	)

	for _, pe := range p.peerMap {
		if _, td := pe.Head(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			if !pe.isSyncing() {
				bestPeer, bestTd = pe, td
			}
		}
	}

	return bestPeer
}

func (p *peerSet) Remove(peerID common.Address) {
	p.lock.Lock()
	defer p.lock.Unlock()

	result := p.peerMap[peerID]
	if result != nil {
		delete(p.peerMap, peerID)
	}
}

func (p *peerSet) Add(pe *peer) {
	p.lock.Lock()
	defer p.lock.Unlock()

	peerID := pe.peerID
	p.peerMap[peerID] = pe
}

func (p *peerSet) Find(address common.Address) *peer {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.peerMap[address]
}

func (p *peerSet) choosePeers(filter peerFilter) (choosePeers []*peer) {
	p.lock.Lock()
	defer p.lock.Unlock()

	mapLen := len(p.peerMap)
	peerL := make([]*peer, mapLen)
	var filteredPeers []*peer

	idx := 0
	for _, v := range p.peerMap {
		peerL[idx] = v
		idx++

		if !filter.blockHash.IsEmpty() && v.findIdxByHash(filter.blockHash) != -1 {
			filteredPeers = append(filteredPeers, v)
		}
	}

	const maxPeers = 3

	// choose filtered peers
	if len := len(filteredPeers); len > 0 {
		if len <= maxPeers {
			return filteredPeers
		}

		perm := rand.Perm(len)
		for i := 0; i < maxPeers; i++ {
			choosePeers = append(choosePeers, filteredPeers[perm[i]])
		}

		return
	}

	common.Shuffle(peerL)
	cnt := 0
	for _, p := range peerL {
		cnt++
		choosePeers = append(choosePeers, p)
		if cnt >= maxPeers {
			return
		}
	}

	return
}
