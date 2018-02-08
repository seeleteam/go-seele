/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"net"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
)

type MsgType uint8

const (
	// add msg type flag at the first byte of the message
	PingMsg      MsgType = 1
	PongMsg      MsgType = 2
	FindNodeMsg  MsgType = 3
	NeighborsMsg MsgType = 5
)

const (
	VERSION uint = 1
)

type ping struct {
	Version uint // TODO add version check
	SelfID  NodeID

	to NodeID
}

type pong struct {
	SelfID NodeID
}

type findNode struct {
	SelfID  NodeID
	QueryID NodeID

	to NodeID
}

type neighbors struct {
	SelfID NodeID
	Nodes  []*rpcNode
}

type rpcNode struct {
	SelfID  NodeID
	IP      net.IP
	UDPPort uint16
}

func (r *rpcNode) ToNode() *Node {
	return NewNode(r.SelfID, r.IP, r.UDPPort)
}

func byteToMsgType(byte byte) MsgType {
	return MsgType(uint8(byte))
}

func msgTypeToByte(t MsgType) byte {
	return byte(t)
}

func generateBuff(code MsgType, encoding []byte) []byte {
	buff := []byte{msgTypeToByte(code)}

	return append(buff, encoding...)
}

// handle send ping msg and add pending
func (m *ping) handle(t *udp, from *net.UDPAddr) {
	log.Debug("received ping from: %s", common.BytesToHex(m.SelfID.Bytes()))

	// response with pong
	if m.Version != VERSION {
		return
	}

	resp := &pong{
		SelfID: t.self.ID,
	}

	t.sendMsg(PongMsg, resp, from)
}

func (m *ping) send(t *udp, to *net.UDPAddr) {
	log.Debug("send ping msg to: %s", common.BytesToHex(m.to.Bytes()))

	p := &pending{
		from: m.to,
		code: PongMsg,

		callback: func(resp interface{}, addr *net.UDPAddr) (done bool) {
			r := resp.(*pong)
			n := NewNodeWithAddr(r.SelfID, addr)
			t.table.updateNode(n)

			log.Debug("received pong msg: %s", common.BytesToHex(r.SelfID.Bytes()))

			return true
		},
		errorCallBack: func() { // delete this node when ping timeout, TODO add time limit
			sha := m.to.ToSha()
			t.table.deleteNode(sha)
			t.db.delete(sha)
		},
	}

	t.addpending <- p
	t.sendMsg(PingMsg, m, to)
}

func (m *findNode) handle(t *udp, from *net.UDPAddr) {
	log.Debug("received find node request from: %s", common.BytesToHex(m.SelfID.Bytes()))

	node := NewNodeWithAddr(m.SelfID, from)
	t.table.AddNode(node)
	t.db.add(node.sha, node)

	// response find node request
	nodes := t.table.findNodeResponse(m.to.ToSha())

	rpcs := make([]*rpcNode, len(nodes))
	for index, n := range nodes {
		rpcs[index] = &rpcNode{
			SelfID:  n.ID,
			IP:      n.IP,
			UDPPort: n.UDPPort,
		}
	}

	response := &neighbors{
		Nodes:  rpcs,
		SelfID: t.self.ID,
	}

	t.sendMsg(NeighborsMsg, response, from)
}

func (m *findNode) send(t *udp, to *net.UDPAddr) {
	log.Debug("send find msg to: %s", common.BytesToHex(m.to.Bytes()))

	p := &pending{
		from: m.to,
		code: NeighborsMsg,

		callback: func(resp interface{}, addr *net.UDPAddr) (done bool) {
			log.Debug("received neighbors")

			r := resp.(*neighbors)

			log.Debug("received neighbors msg from: %s", common.BytesToHex(r.SelfID.Bytes()))

			if r.Nodes == nil || len(r.Nodes) == 0 {
				return true
			}

			found := false
			for _, n := range r.Nodes {
				if n.SelfID == m.to {
					found = true
				}

				node := n.ToNode()
				t.table.AddNode(node)
				t.db.add(node.sha, node)
			}

			if !found {
				nodes := t.table.findNodeForRequest(m.to.ToSha())
				sendFindNodeRequest(t, nodes, m.to)
			}

			return true
		},
		errorCallBack: func() {
			// do nothing
		},
	}

	t.addpending <- p
	t.sendMsg(FindNodeMsg, m, to)
}

func sendFindNodeRequest(u *udp, nodes []*Node, target NodeID) {
	if nodes == nil || len(nodes) == 0 {
		return
	}

	for _, n := range nodes {
		f := &findNode{
			SelfID:  u.self.ID,
			QueryID: target,
			to:      n.ID,
		}

		addr := &net.UDPAddr{
			IP:   n.IP,
			Port: int(n.UDPPort),
		}

		f.send(u, addr)
	}
}
