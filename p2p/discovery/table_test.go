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

	log := log.GetLogger("discovery", common.LogConfig.PrintLog)
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
