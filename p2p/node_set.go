/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/log"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

// nodeItem represents node information
type nodeItem struct {
	node       *discovery.Node
	bConnected bool // whether node is connected or not
}

// nodeSet is thread safe collection, contains all active nodes, weather it is connected or not
type nodeSet struct {
	lock    sync.RWMutex
	nodeMap map[common.Address]*nodeItem
	ipSet   map[uint]map[string]uint
	log     *log.SeeleLog
}

// NewNodeSet creates new nodeSet
func NewNodeSet() *nodeSet {
	rand.Seed(time.Now().UnixNano())
	ipSet := make(map[uint]map[string]uint)
	for i := uint(1); i <= common.ShardCount; i++ {
		ipSet[i] = make(map[string]uint)
	}
	return &nodeSet{
		nodeMap: make(map[common.Address]*nodeItem),
		lock:    sync.RWMutex{},
		ipSet:   ipSet,
		log:     log.GetLogger("p2p"),
	}
}

func (set *nodeSet) getSelfShardNodeNum() int {
	count := 0
	for _, item := range set.nodeMap {
		if item.node.Shard == common.LocalShardNumber && item.bConnected {
			count++
		}
	}
	return count
}
func (set *nodeSet) setNodeStatus(p *discovery.Node, bConnected bool) {
	set.lock.Lock()
	defer set.lock.Unlock()

	item := set.nodeMap[p.ID]
	if item == nil {
		return
	}

	item.bConnected = bConnected
}

// tryAdd add a new node to the map if it's not exist.
func (set *nodeSet) tryAdd(p *discovery.Node) {
	set.lock.Lock()
	defer set.lock.Unlock()

	if set.nodeMap[p.ID] != nil {
		return
	}
	// Ignore node if nodes from same ip reach max limit
	if set.ipSet != nil {
		nodeCnt, _ := set.ipSet[p.Shard][p.IP.String()]
		if nodeCnt > maxConnsPerShardPerIp {
			set.log.Warn("tryAdd a new node. Reached connection limit for single IP, node:%v", p.String())
			return
		}
	}
	item := &nodeItem{
		node:       p,
		bConnected: false,
	}
	set.nodeMap[p.ID] = item
	if _, ok := set.ipSet[p.Shard][p.IP.String()]; ok {
		set.ipSet[p.Shard][p.IP.String()]++
	} else {
		set.ipSet[p.Shard][p.IP.String()] = 1
	} // add ip count
}

func (set *nodeSet) delete(p *discovery.Node) {
	set.lock.Lock()
	defer set.lock.Unlock()

	delete(set.nodeMap, p.ID)
	if _, ok := set.ipSet[p.Shard][p.IP.String()]; ok {
		set.ipSet[p.Shard][p.IP.String()]-- //update ip count
	} else {
		fmt.Println("no IP found to delete")
	}

}

// randSelect select one node randomly from nodeMap which is not connected yet
func (set *nodeSet) randSelect() []*discovery.Node {
	set.lock.RLock()
	defer set.lock.RUnlock()

	var nodeL []*discovery.Node
	var retNodes []*discovery.Node
	nodeCount := make([]int, common.ShardCount)
	for _, v := range set.nodeMap {
		if v.bConnected {
			continue
		}

		nodeL = append(nodeL, v.node)
	}

	if len(nodeL) == 0 {
		return nil
	}

	perm := rand.Perm(len(nodeL))
	k := 0
	for i := 0; i < len(nodeL); i++ {
		if nodeCount[nodeL[perm[i]].Shard-1] < 1 {
			nodeCount[nodeL[perm[i]].Shard-1]++
			retNodes = append(retNodes, nodeL[perm[i]])
			k++
		}
		if k >= common.ShardCount {
			break
		}
	}
	return retNodes
}
