/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

type msgType uint8

const (
	// add msg type flag at the first byte of the message
	pingMsgType      msgType = 1
	pongMsgType      msgType = 2
	findNodeMsgType  msgType = 3
	neighborsMsgType msgType = 4
	findShardNodeMsgType msgType = 5
	shardNodeMsgType msgType = 6
)

const (
	discoveryProtocolVersion uint = 1
)

type ping struct {
	Version uint // TODO add version check
	SelfID  common.Address
	Shard   uint

	to *Node
}

type pong struct {
	SelfID common.Address
	Shard uint
}

type findNode struct {
	SelfID  common.Address
	QueryID common.Address // the ID we want to query in Kademila

	to *Node // the node that send request to
}

type neighbors struct {
	SelfID common.Address
	Nodes  []*rpcNode
}

type findShardNode struct {
	SelfID common.Address
	Shard uint

	to *Node
}

type shardNode struct {
	SelfID common.Address
	Nodes []*rpcNode
}

type rpcNode struct {
	SelfID  common.Address
	IP      net.IP
	UDPPort uint16
	Shard 	uint
}

func (r *rpcNode) ToNode() *Node {
	return NewNode(r.SelfID, r.IP, int(r.UDPPort), int(r.Shard))
}

func ConvertToRpcNode(n *Node) *rpcNode {
	return &rpcNode{
		SelfID:  n.ID,
		IP:      n.IP,
		UDPPort: uint16(n.UDPPort),
		Shard:uint(n.Shard),
	}
}

func byteToMsgType(byte byte) msgType {
	return msgType(uint8(byte))
}

func msgTypeToByte(t msgType) byte {
	return byte(t)
}

func generateBuff(code msgType, encoding []byte) []byte {
	buff := []byte{msgTypeToByte(code)}

	return append(buff, encoding...)
}

// handle send pong msg and add pending
func (m *ping) handle(t *udp, from *net.UDPAddr) {
	t.log.Debug("received ping from: %s", m.SelfID.ToHex())

	// response with pong
	if m.Version != discoveryProtocolVersion {
		return
	}

	resp := &pong{
		SelfID: t.self.ID,
	}

	t.sendMsg(pongMsgType, resp, NewNodeWithAddr(m.SelfID, from, int(m.Shard)))
}

// send send ping message and handle callback
func (m *ping) send(t *udp) {
	t.log.Debug("send ping msg to: %s", m.to.ID.ToHex())

	p := &pending{
		from: m.to,
		code: pongMsgType,

		callback: func(resp interface{}, addr *net.UDPAddr) (done bool) {
			r := resp.(*pong)
			n := NewNodeWithAddr(r.SelfID, addr, int(r.Shard))
			t.table.updateNode(n)

			t.log.Debug("received pong msg: %s", r.SelfID.ToHex())

			return true
		},
		errorCallBack: func() { // delete this node when ping timeout, TODO add time limit
			t.deleteNode(m.to)
		},
	}

	t.addPending <- p
	t.sendMsg(pingMsgType, m, m.to)
}

// handle response find node request
func (m *findNode) handle(t *udp, from *net.UDPAddr) {
	t.log.Debug("received find node request from: %s", m.SelfID.ToHex())
	node := NewNodeWithAddr(m.SelfID, from)
	t.addNode(node)

	nodes := t.table.findNodeWithTarget(crypto.HashBytes(m.QueryID.Bytes()))

	rpcs := make([]*rpcNode, len(nodes))
	for index, n := range nodes {
		rpcs[index] = &rpcNode{
			SelfID:  n.ID,
			IP:      n.IP,
			UDPPort: uint16(n.UDPPort),
		}
	}

	response := &neighbors{
		Nodes:  rpcs,
		SelfID: t.self.ID,
	}

	t.sendMsg(neighborsMsgType, response, NewNodeWithAddr(m.SelfID, from))
}

// send send find node message and handle callback
func (m *findNode) send(t *udp) {
	t.log.Debug("send find msg to: %s", m.to.ID.ToHex())

	p := &pending{
		from: m.to,
		code: neighborsMsgType,

		callback: func(resp interface{}, addr *net.UDPAddr) (done bool) {
			r := resp.(*neighbors)

			t.log.Debug("received neighbors msg from: %s", r.SelfID.ToHex())
			if r.Nodes == nil || len(r.Nodes) == 0 {
				return true
			}

			t.log.Debug("got find response with %d nodes", len(r.Nodes))

			found := false
			for _, n := range r.Nodes {
				t.log.Debug("received node: %s", n.SelfID.ToHex())

				if n.SelfID == m.QueryID {
					found = true
				}

				node := n.ToNode()
				t.addNode(node)
			}

			// if not found, will find the node that is more closer than last one
			if !found {
				nodes := t.table.findNodeWithTarget(crypto.HashBytes(m.QueryID.Bytes()))
				sendFindNodeRequest(t, nodes, m.QueryID)
			}

			return true
		},
		errorCallBack: func() {
			// do nothing
		},
	}

	t.addPending <- p
	t.sendMsg(findNodeMsgType, m, m.to)
}

func sendFindNodeRequest(u *udp, nodes []*Node, target common.Address) {
	if nodes == nil || len(nodes) == 0 {
		return
	}

	for _, n := range nodes {
		f := &findNode{
			SelfID:  u.self.ID,
			QueryID: target,
			to:      n,
		}

		f.send(u)
	}
}

func (m *findShardNode) send(t *udp) {
	t.log.Debug("send find shard node msg to: %s", m.to.ID.ToHex())

	p := &pending{
		from: m.to,
		code: shardNodeMsgType,

		callback: func(resp interface{}, addr *net.UDPAddr) (done bool) {
			r := resp.(*shardNode)
			n := NewNodeWithAddr(r.SelfID, addr)
			t.table.updateNode(n)

			for _, node := range r.Nodes {
				t.table.addNode(node.ToNode())
			}

			return true
		},
		errorCallBack: func() { // delete this node when ping timeout, TODO add time limit
			t.deleteNode(m.to)
		},
	}

	t.addPending <- p
	t.sendMsg(findShardNodeMsgType, m, m.to)
}

func (m *findShardNode) handle(t *udp) {
	bucket := t.table.shardBuckets[m.Shard]
	nodes := bucket.getRandNodes(responseNodeNumber)

	rpcnodes := make([]*rpcNode, len(nodes))
	for i := 0; i < len(nodes); i++ {
		rpcnodes[i] = &rpcNode{
			SelfID:  nodes[i].ID,
			IP:      nodes[i].IP,
			UDPPort: uint16(nodes[i].UDPPort),

		}
	}

	response := shardNode{
		SelfID:t.self.ID,
		Nodes: rpcnodes,
	}

}


