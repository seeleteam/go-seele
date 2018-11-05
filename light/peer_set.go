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

func (p *peerSet) ForEach(handle func(*peer) bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, v := range p.peerMap {
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

func (p *peerSet) choosePeers(shard uint) (choosePeers []*peer) {
	p.lock.Lock()
	defer p.lock.Unlock()

	mapLen := len(p.peerMap)
	peerL := make([]*peer, mapLen)

	idx := 0
	for _, v := range p.peerMap {
		peerL[idx] = v
		idx++
	}

	common.Shuffle(peerL)
	cnt := 0
	for _, p := range peerL {
		cnt++
		choosePeers = append(choosePeers, p)
		if cnt >= 3 {
			return
		}
	}

	return
}
