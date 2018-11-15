/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"encoding/hex"
	"fmt"
	"net"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/stretchr/testify/assert"
)

func Test_Message_CodeToStr(t *testing.T) {
	// valid codes
	code := pingMsgType
	assert.Equal(t, codeToStr(code), "pingMsgType")

	code = pongMsgType
	assert.Equal(t, codeToStr(code), "pongMsgType")

	code = findNodeMsgType
	assert.Equal(t, codeToStr(code), "findNodeMsgType")

	code = neighborsMsgType
	assert.Equal(t, codeToStr(code), "neighborsMsgType")

	code = findShardNodeMsgType
	assert.Equal(t, codeToStr(code), "findShardNodeMsgType")

	code = shardNodeMsgType
	assert.Equal(t, codeToStr(code), "shardNodeMsgType")

	// invalid codes
	code = pingMsgType - 1
	assert.Equal(t, codeToStr(code), "unkwown")

	code = shardNodeMsgType + 1
	assert.Equal(t, codeToStr(code), "unkwown")
}

func Test_Message_ToNode(t *testing.T) {
	r := testRPCNode()
	node := r.ToNode()

	assert.Equal(t, node.ID, r.SelfID)
	assert.Equal(t, node.IP, r.IP)
	assert.Equal(t, node.UDPPort, int(r.UDPPort))
	assert.Equal(t, node.Shard, r.Shard)
}

func Test_Message_ConvertToRPCNode(t *testing.T) {
	r := testRPCNode()
	node := r.ToNode()
	r1 := convertToRPCNode(node)

	assert.Equal(t, r.SelfID, r1.SelfID)
	assert.Equal(t, r.IP, r1.IP)
	assert.Equal(t, r.UDPPort, r1.UDPPort)
	assert.Equal(t, r.Shard, r1.Shard)
}

func Test_Message_ByteToMsgType(t *testing.T) {
	// valid byte codes
	b := byte(1)
	assert.Equal(t, byteToMsgType(b), pingMsgType)

	b = byte(2)
	assert.Equal(t, byteToMsgType(b), pongMsgType)

	b = byte(3)
	assert.Equal(t, byteToMsgType(b), findNodeMsgType)

	b = byte(4)
	assert.Equal(t, byteToMsgType(b), neighborsMsgType)

	b = byte(5)
	assert.Equal(t, byteToMsgType(b), findShardNodeMsgType)

	b = byte(6)
	assert.Equal(t, byteToMsgType(b), shardNodeMsgType)

	// invalid byte codes
	b = byte(0)
	assert.Equal(t, byteToMsgType(b), msgType(0))

	b = byte(7)
	assert.Equal(t, byteToMsgType(b), msgType(7))
}

func Test_Message_Ping_Handle(t *testing.T) {
	p := testPing()
	udp := newTestUDP()
	from, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9666")

	p.handle(udp, from)
	receivedMsg := <-udp.writer
	assert.Equal(t, receivedMsg.code, pongMsgType)

	// invalid version
	p.Version = discoveryProtocolVersion + 1
	p.handle(udp, from)
	assert.Equal(t, true, true) // do nothing and silent
}

func Test_Message_Ping_Send(t *testing.T) {
	p := testPing()
	udp := newTestUDP()
	net.ResolveUDPAddr("udp", "127.0.0.1:8080")

	p.send(udp)
	receivedMsg := <-udp.writer
	assert.Equal(t, receivedMsg.code, pingMsgType)
}

func Test_Message_FindNode_Handle(t *testing.T) {
	f := testFindNode()
	udp := newTestUDP()
	udp.table = testTable()
	from, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9666")

	f.handle(udp, from)
	receivedMsg := <-udp.writer
	assert.Equal(t, receivedMsg.code, neighborsMsgType)
}

func Test_Message_FindNode_Send(t *testing.T) {
	f := testFindNode()
	udp := newTestUDP()
	net.ResolveUDPAddr("udp", "127.0.0.1:8080")

	f.send(udp)
	receivedMsg := <-udp.writer
	assert.Equal(t, receivedMsg.code, findNodeMsgType)
}

func Test_Message_FindShardNode_Handle(t *testing.T) {
	fs := testFindShardNode()
	udp := newTestUDP()
	udp.table = testTable()
	from, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9666")

	fs.handle(udp, from)
	receivedMsg := <-udp.writer
	assert.Equal(t, receivedMsg.code, shardNodeMsgType)
}

func Test_Message_FindShardNode_Send(t *testing.T) {
	fs := testFindShardNode()
	udp := newTestUDP()
	net.ResolveUDPAddr("udp", "127.0.0.1:8080")

	fs.send(udp)
	receivedMsg := <-udp.writer
	assert.Equal(t, receivedMsg.code, findShardNodeMsgType)
}

func Test_Message_SendFindNodeRequest(t *testing.T) {
	id, _ := crypto.GenerateRandomAddress()
	udp := newTestUDP()
	udp.table = testTable()

	sendFindNodeRequest(udp, nil, *id)
	assert.Equal(t, true, true) // silent
}

func testRPCNode() *rpcNode {
	r := &rpcNode{
		SelfID:  common.HexMustToAddres("0xd0c549b022f5a17a8f50a4a448d20ba579d01781"),
		IP:      net.IPv4(127, 0, 0, 1),
		UDPPort: 8080,
		Shard:   1,
	}

	return r
}

func testPing() *ping {
	r := testRPCNode()
	node := r.ToNode()

	p := &ping{
		Version:   discoveryProtocolVersion,
		SelfID:    common.HexMustToAddres("0xd0c549b022f5a17a8f50a4a448d20ba579d01781"),
		SelfShard: 1,
		to:        node,
	}

	return p
}

func testFindNode() *findNode {
	r := testRPCNode()
	node := r.ToNode()

	f := &findNode{
		SelfID:  common.HexMustToAddres("0xd0c549b022f5a17a8f50a4a448d20ba579d01781"),
		QueryID: common.HexMustToAddres("0xbc495ea1980db8a2451ece7708c29c12caa9c071"),
		to:      node,
	}

	return f
}

func testFindShardNode() *findShardNode {
	r := testRPCNode()
	node := r.ToNode()

	fs := &findShardNode{
		SelfID:       common.HexMustToAddres("0xd0c549b022f5a17a8f50a4a448d20ba579d01781"),
		RequestShard: 1,
		to:           node,
	}

	return fs
}

func testTable() *Table {
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
	table.addNode(node1)
	table.addNode(node2)

	return table
}
