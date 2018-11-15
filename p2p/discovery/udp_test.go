package discovery

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/orcaman/concurrent-map"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/stretchr/testify/assert"
)

var (
	selfNode = MustNewNodeWithAddr(*crypto.MustGenerateShardAddress(1), "127.0.0.1:9666", 1)
)

func newTestUDP() *udp {
	node1 := MustNewNodeWithAddr(*crypto.MustGenerateShardAddress(1), "127.0.0.1:9000", 1)
	node2 := MustNewNodeWithAddr(*crypto.MustGenerateShardAddress(1), "127.0.0.1:9888", 1)

	log := log.GetLogger("discovery")
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9666")
	return &udp{
		trustNodes:        []*Node{node1, node2},
		table:             newTable(selfNode.ID, addr, 1, log),
		self:              NewNodeWithAddr(selfNode.ID, addr, 1),
		db:                NewDatabase(log),
		writer:            make(chan *send, 1),
		addPending:        make(chan *pending, 1),
		log:               log,
		timeoutNodesCount: cmap.New(),
	}
}

func Test_UDP_NewUDP(t *testing.T) {
	id := *crypto.MustGenerateShardAddress(1)
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9666")

	udp := newUDP(id, addr, 0)
	assert.Equal(t, udp != nil, true)
	assert.Equal(t, udp.self, NewNodeWithAddr(id, addr, 0))
	assert.Equal(t, udp.localAddr, addr)
}

type testStruct struct {
	data int64
}

func Test_UDP_SendMsg(t *testing.T) {
	udp := newTestUDP()
	assert.Equal(t, udp != nil, true)

	toID := *crypto.MustGenerateShardAddress(1)
	toAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9666")
	udp.sendMsg(pingMsgType, &testStruct{1}, toID, toAddr)

	receivedMsg := <-udp.writer
	assert.Equal(t, receivedMsg.code, pingMsgType)
	assert.Equal(t, receivedMsg.toID, toID)
	assert.Equal(t, receivedMsg.toAddr, toAddr)
}

func Test_UDP_SendConnMsg(t *testing.T) {
	udp := newTestUDP()
	assert.Equal(t, udp != nil, true)

	toAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9667")
	conn, _ := getUDPConn(toAddr)
	result := udp.sendConnMsg([]byte("testmsg"), conn, toAddr)
	assert.Equal(t, result, true)

	// failed to send message due to invalid ip address
	toAddr, _ = net.ResolveUDPAddr("udp", "")
	result = udp.sendConnMsg([]byte("testmsg"), conn, toAddr)
	assert.Equal(t, result, false)
}

func Test_UDP_AddNode(t *testing.T) {
	u := newTestUDP()
	assert.Equal(t, u.db.size(), 0)

	// add normal node
	u.addNode(u.trustNodes[0], false)
	assert.Equal(t, u.db.size(), 1)

	// add self
	u.addNode(selfNode, false)
	assert.Equal(t, u.db.size(), 1)

	// add duplicated node
	u.addNode(u.trustNodes[0], false)
	assert.Equal(t, u.db.size(), 1)
}

func Test_UDP_DeleteNode(t *testing.T) {
	u := newTestUDP()
	assert.Equal(t, u.db.size(), 0)

	// add and then delete, just insert this node into map timeoutNodesCount,
	// this node will be deleted until the number of deletion operation is equal to timeoutCountForDeleteNode
	node := MustNewNodeWithAddr(*crypto.MustGenerateShardAddress(1), "127.0.0.1:9881", 1)
	u.addNode(node, false)
	assert.Equal(t, u.db.size(), 1)

	for i := 1; i < timeoutCountForDeleteNode; i++ {
		u.deleteNode(node)
		assert.Equal(t, u.db.size(), 1)
	}
	u.deleteNode(node)
	assert.Equal(t, u.db.size(), 0)

	// delete self node
	u.deleteNode(selfNode)
	assert.Equal(t, u.db.size(), 0)

	// delete nonexistent node
	nonexistent := MustNewNodeWithAddr(*crypto.MustGenerateShardAddress(1), "127.0.0.1:9889", 1)
	u.deleteNode(nonexistent)
	assert.Equal(t, u.db.size(), 0)
}

func Test_UDP_LoadNodes(t *testing.T) {
	tempFolder := common.GetTempFolder()

	u := newTestUDP()
	u.addNode(u.trustNodes[0], false)
	u.addNode(u.trustNodes[1], false)
	u.db.SaveNodes(tempFolder)
	defer func() {
		if fileFullPath := filepath.Join(tempFolder, NodesBackupFileName); common.FileOrFolderExists(fileFullPath) {
			os.Remove(fileFullPath)
		}
	}()

	assert.Equal(t, len(u.bootstrapNodes), 0)
	u.loadNodes(tempFolder)
	assert.Equal(t, len(u.bootstrapNodes), 2)

	// nodes folder doesn't exist
	log := log.GetLogger("discovery")
	u = &udp{
		log: log,
	}

	u.loadNodes("nonexistentfolder")
	assert.Equal(t, len(u.bootstrapNodes), 0)
}
