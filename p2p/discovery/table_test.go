package discovery

import (
	"encoding/hex"
	"fmt"
	"net"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
)

func newTestTable() *Table {
	selfID := "snode://0101f3c956d0a320b153a097c3d04efa48888888@127.0.0.1:9666[1]"
	self, err := NewNodeFromString(selfID)
	if err != nil {
		panic(err)
	}

	log := log.GetLogger("discovery")
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9666")
	return newTable(self.ID, addr, 1, log)
}

func Test_addNode(t *testing.T) {
	id1 := "snode://0101f3c956d0a320b153a097c3d04efa488d43d7@127.0.0.1:9000[1]"
	node1, err := NewNodeFromString(id1)
	if err != nil {
		panic(err)
	}

	add2 := common.HexMustToAddres("0xe58010916a17a5d333814f8bae82db6cb6b7ab81")
	id2 := fmt.Sprintf("snode://%v%v", hex.EncodeToString(add2.Bytes()), "@127.0.0.1:9888[2]")
	node2, err := NewNodeFromString(id2)
	if err != nil {
		panic(err)
	}

	table := newTestTable()
	assert.Equal(t, len(table.buckets) == nBuckets, true)
	assert.Equal(t, len(table.shardBuckets) == common.ShardCount+1, true)

	table.addNode(node1)
	dis := logDist(table.selfNode.getSha(), node1.getSha())
	assert.Equal(t, len(table.buckets[dis].peers) == 1, true)
	assert.Equal(t, len(table.buckets[dis-1].peers) != 1, true)
	assert.Equal(t, len(table.buckets[dis-2].peers) != 1, true)
	assert.Equal(t, table.buckets[dis].peers[0] == node1, true)

	table.addNode(node2)
	assert.Equal(t, len(table.shardBuckets[2].peers) == 1, true)
	assert.Equal(t, len(table.shardBuckets[1].peers) != 1, true)
	assert.Equal(t, len(table.shardBuckets[3].peers) != 1, true)
	assert.Equal(t, table.shardBuckets[2].peers[0] == node2, true)
}

func Test_findNodeWithTarget(t *testing.T) {
	id1 := "snode://0101f3c956d0a320b153a097c3d04efa488d43d7@127.0.0.1:9000[1]"
	node1, err := NewNodeFromString(id1)
	assert.Equal(t, err, nil)

	add2 := common.HexMustToAddres("0xe58010916a17a5d333814f8bae82db6cb6b7ab81")
	id2 := fmt.Sprintf("snode://%v%v", hex.EncodeToString(add2.Bytes()), "@127.0.0.1:9888[2]")
	node2, err := NewNodeFromString(id2)
	assert.Equal(t, err, nil)

	table := newTestTable()
	table.addNode(node1)
	table.addNode(node2)

	nodes := table.findNodeWithTarget(node1.getSha())
	assert.Equal(t, len(nodes), 1)
	assert.Equal(t, nodes[0] == node1, true)

	//The shard of the table is 1; the nodes of shard 1 will return, if the distance is less than target that comparing with the table self node
	nodes2 := table.findNodeWithTarget(node2.getSha())
	assert.Equal(t, len(nodes2), 1)
	assert.Equal(t, nodes2[0] != node2, true)
	assert.Equal(t, nodes2[0] == node1, true)

	//The nodes of shard 1 will return, becaus of the distance is less than target that comparing with the table selnode
	noExistKey := common.HexMustToAddres("0x2a87b6504cd00af95a83b9887112016a2a991cf1")
	noExistId := fmt.Sprintf("snode://%v%v", hex.EncodeToString(noExistKey.Bytes()), "@127.0.0.1:9888[1]")
	noExistNode, err := NewNodeFromString(noExistId)
	nodes1 := table.findNodeWithTarget(noExistNode.getSha())
	assert.Equal(t, len(nodes1), 1)

	//The nodes of shard 1 willn't return, becaus of the distance is greater than target that comparing with the table selnode
	noExistKey2 := common.HexMustToAddres("0xfbe506bdaf256682551873290d0a794d51bac4d1")
	noExistId2 := fmt.Sprintf("snode://%v%v", hex.EncodeToString(noExistKey2.Bytes()), "@127.0.0.1:9888[2]")
	noExistNode2, err := NewNodeFromString(noExistId2)
	nodes2 = table.findNodeWithTarget(noExistNode2.getSha())
	assert.Equal(t, len(nodes2), 0)
}
