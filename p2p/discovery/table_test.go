package discovery

import (
	"encoding/hex"
	"fmt"
	"net"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	"github.com/stretchr/testify/assert"
)

func newTestTable() *Table {
	selfID := "snode://0101f3c956d0a320b153a097c3d04efa48888881@127.0.0.1:9666[1]"
	self, err := NewNodeFromString(selfID)
	if err != nil {
		panic(err)
	}

	log := log.GetLogger("discovery")
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9666")
	return newTable(self.ID, addr, 1, log)
}

func Test_addNode(t *testing.T) {
	id1 := "snode://0101f3c956d0a320b153a097c3d04efa488d43d1@127.0.0.1:9000[1]"
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
	assert.Equal(t, table.shardBuckets[2].peers[0] == node2, true)
}

func Test_findNodeWithTarget(t *testing.T) {
	id1 := "snode://0101f3c956d0a320b153a097c3d04efa488d43d1@127.0.0.1:9000[1]"
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
	assert.Equal(t, len(nodes2), 0)

	//The nodes of shard 1 will return, becaus of the distance is less than target that comparing with the table selnode
	noExistKey := common.HexMustToAddres("0x2a87b6504cd00af95a83b9887112016a2a991cf1")
	noExistID := fmt.Sprintf("snode://%v%v", hex.EncodeToString(noExistKey.Bytes()), "@127.0.0.1:9888[1]")
	noExistNode, err := NewNodeFromString(noExistID)
	nodes1 := table.findNodeWithTarget(noExistNode.getSha())
	assert.Equal(t, len(nodes1), 1)

	//The nodes of shard 1 won't return, because of the distance is greater than target that comparing with the table selnode
	noExistKey2 := common.HexMustToAddres("0xfbe506bdaf256682551873290d0a794d51bac4d1")
	noExistID2 := fmt.Sprintf("snode://%v%v", hex.EncodeToString(noExistKey2.Bytes()), "@127.0.0.1:9888[2]")
	noExistNode2, err := NewNodeFromString(noExistID2)
	nodes2 = table.findNodeWithTarget(noExistNode2.getSha())
	assert.Equal(t, len(nodes2), 0)
}

func Test_deleteNode(t *testing.T) {
	id1 := "snode://0101f3c956d0a320b153a097c3d04efa488d43d1@127.0.0.1:9000[1]"
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
	table.deleteNode(node1)
	nodes = table.findNodeWithTarget(node1.getSha())
	assert.Equal(t, len(nodes), 0)
	assert.Equal(t, len(table.shardBuckets[2].peers), 1)

	noExistKey := common.HexMustToAddres("0x2a87b6504cd00af95a83b9887112016a2a991cf1")
	noExistID := fmt.Sprintf("snode://%v%v", hex.EncodeToString(noExistKey.Bytes()), "@127.0.0.1:9888[1]")
	noExistNode, err := NewNodeFromString(noExistID)
	table.deleteNode(noExistNode)
	assert.Equal(t, len(nodes), 0)
	assert.Equal(t, len(table.shardBuckets[2].peers), 1)

	table.deleteNode(node2)
	assert.Equal(t, len(nodes), 0)
	assert.Equal(t, len(table.shardBuckets[2].peers), 0)
}

func Test_GetRandNodes(t *testing.T) {
	id1 := "snode://0101f3c956d0a320b153a097c3d04efa488d43d1@127.0.0.1:9000[1]"
	node1, err := NewNodeFromString(id1)
	assert.Equal(t, err, nil)

	add2 := common.HexMustToAddres("0xe58010916a17a5d333814f8bae82db6cb6b7ab81")
	id2 := fmt.Sprintf("snode://%v%v", hex.EncodeToString(add2.Bytes()), "@127.0.0.1:9888[2]")
	node2, err := NewNodeFromString(id2)
	assert.Equal(t, err, nil)

	table := newTestTable()
	table.addNode(node1)
	table.addNode(node2)

	nodes := table.GetRandNodes(0)
	assert.Equal(t, len(nodes), 0)
	nodes = table.GetRandNodes(1)
	assert.Equal(t, len(nodes), 1)
	nodes = table.GetRandNodes(2)
	assert.Equal(t, len(nodes), 1)

	add11 := common.HexMustToAddres("0x4fb7c8b0287378f0cf8b5a9262bf3ef7e101f8d1")
	id11 := fmt.Sprintf("snode://%v%v", hex.EncodeToString(add11.Bytes()), "@127.0.0.1:9888[1]")
	node11, err := NewNodeFromString(id11)
	assert.Equal(t, err, nil)
	table.addNode(node11)
	nodes = table.GetRandNodes(2)
	assert.Equal(t, len(nodes), 2)
}
