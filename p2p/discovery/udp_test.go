package discovery

import (
	"net"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

func Test_addTrustNodes(t *testing.T) {
	id1 := "snode://0101f3c956d0a320b153a097c3d04efa488d43d7d7e05a44791492c9979ff558@192.168.122.132:9000[1]"
	node1, err := NewNodeFromString(id1)
	assert.Equal(t, err, nil)

	id2 := "snode://0101f3c956d0a320b153a097c3d04efa488d43d7d7e05a44791492c9979ff588@192.168.122.132:9888[1]"
	node2, err := NewNodeFromString(id2)
	assert.Equal(t, err, nil)

	selfId := "snode://0101f3c956d0a320b153a097c3d04efa488d43d7d7e05a44791492c9979ff566@192.168.122.132:9666[1]"
	self, err := NewNodeFromString(selfId)
	assert.Equal(t, err, nil)

	log := log.GetLogger("discovery", common.LogConfig.PrintLog)
	addr, err := net.ResolveUDPAddr("udp", "192.168.122.132:9666")
	u := &udp{
		trustNodes: []*Node{node1, node2},
		table:      newTable(self.ID, addr, 1, log),
		self:       NewNodeWithAddr(self.ID, addr, 1),

		db:  NewDatabase(log),
		log: log,
	}

	u.addTrustNodes()
	assert.Equal(t, len(u.db.m), 2)

	key1 := crypto.HashBytes(node1.ID.Bytes())
	assert.Equal(t, u.db.m[key1].UDPPort, 9000)

	key2 := crypto.HashBytes(node2.ID.Bytes())
	assert.Equal(t, u.db.m[key2].UDPPort, 9888)

	u.addTrustNodes()
	assert.Equal(t, len(u.db.m), 2)
}
