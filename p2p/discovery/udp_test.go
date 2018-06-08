package discovery

import (
	"net"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
)

func newTestUdp() *udp {
	id1 := "snode://0101f3c956d0a320b153a097c3d04efa488d43d7@127.0.0.1:9000[1]"
	node1, err := NewNodeFromString(id1)
	if err != nil {
		panic(err)
	}

	id2 := "snode://0101f3c956d0a320b153a097c3d04efa488d6666@127.0.0.1:9888[1]"
	node2, err := NewNodeFromString(id2)
	if err != nil {
		panic(err)
	}

	selfId := "snode://0101f3c956d0a320b153a097c3d04efa48888888@127.0.0.1:9666[1]"
	self, err := NewNodeFromString(selfId)
	if err != nil {
		panic(err)
	}

	log := log.GetLogger("discovery", common.LogConfig.PrintLog)
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9666")
	return &udp{
		trustNodes: []*Node{node1, node2},
		table:      newTable(self.ID, addr, 1, log),
		self:       NewNodeWithAddr(self.ID, addr, 1),

		db:  NewDatabase(log),
		log: log,
	}
}

func Test_addTrustNodes(t *testing.T) {
	u := newTestUdp()

	u.addTrustNodes()
	assert.Equal(t, len(u.db.m), 2)

	u.addTrustNodes()
	assert.Equal(t, len(u.db.m), 2)
}

func Test_loadNodes(t *testing.T) {
	u := newTestUdp()
	u.addTrustNodes()
	u.db.SaveNodes(common.GetTempFolder())
	for k, _ := range u.db.m {
		u.db.delete(k)
	}

	assert.Equal(t, len(u.db.m), 0)
	u.loadNodes(common.GetTempFolder())
	assert.Equal(t, len(u.db.m), 2)
}
