/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

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
	var bestPeer *peer
	var bestHash common.Hash
	bestTd := big.NewInt(0)	

	peers := p.getPeerByShard(shard)
	for _, peer := range peers {
		// if the total difficulties of the peers are the same, compare their head hashes
		if hash, td := peer.Head(); bestPeer == nil || td.Cmp(bestTd) > 0 || (td.Cmp(bestTd) == 0 && hash.Big().Cmp(bestHash.Big()) > 0) {
			bestPeer, bestTd, bestHash = peer, td, hash
		}
	}

	return bestPeer
}

func (p *peerSet) bestPeers(shard uint) []*peer {
	var bestPeers []*peer
	
	peers := p.getPeerByShard(shard)

	peersMap := make(map[common.Address]bool)
	// the number of best peers
	NumOfBestPeers := 3
	if len(peers) < NumOfBestPeers {
		NumOfBestPeers = len(peers)
	}
	i := 0
	for i < NumOfBestPeers {
		bestTd := big.NewInt(0)
		var bestPeer *peer
		for _, peer := range peers {
			if peersMap[peer.peerID] == true {
				continue
			}			
			if _, td := peer.Head(); bestPeer == nil || td.Cmp(bestTd) > 0 {
				bestPeer, bestTd = peer, td
			} 
		}
		bestPeers = append(bestPeers, bestPeer)
		peersMap[bestPeer.peerID] = true
		i++
	}
	return bestPeers

}

func (p *peerSet) Find(address common.Address) *peer {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.peerMap[address]
}

func (p *peerSet) Remove(address common.Address) {
	p.lock.Lock()
	defer p.lock.Unlock()

	result := p.peerMap[address]
	if result != nil {
		delete(p.peerMap, address)
		delete(p.shardPeers[result.Node.Shard], address)
	}
}

func (p *peerSet) Add(pe *peer) {
	p.lock.Lock()
	defer p.lock.Unlock()

	address := pe.Node.ID
	result := p.peerMap[address]
	if result != nil {
		delete(p.peerMap, address)
		delete(p.shardPeers[result.Node.Shard], address)
	}

	p.peerMap[address] = pe
	p.shardPeers[pe.Node.Shard][address] = pe
}

func (p *peerSet) getAllPeers() []*peer {
	p.lock.RLock()
	defer p.lock.RUnlock()

	value := make([]*peer, len(p.peerMap))
	index := 0
	for _, v := range p.peerMap {
		value[index] = v
		index++
	}

	return value
}

func (p *peerSet) getPeerByShard(shard uint) []*peer {
	p.lock.RLock()
	defer p.lock.RUnlock()

	value := make([]*peer, len(p.shardPeers[shard]))
	index := 0
	for _, v := range p.shardPeers[shard] {
		value[index] = v
		index++
	}

	return value
}

func (p *peerSet) getPeerCountByShard(shard uint) int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return len(p.shardPeers[shard])
}
