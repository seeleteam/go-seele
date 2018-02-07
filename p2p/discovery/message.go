/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"net"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/common"
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
	ID NodeID

	target NodeID
}

type pong struct {
	ID NodeID
}

type findNode struct {
	ID NodeID
	target NodeID
}

type neighbors struct {
	ID NodeID
	Nodes []*rpcNode
}

type rpcNode struct {
	ID      NodeID
	IP      net.IP
	UDPPort uint16
}

func (r *rpcNode) ToNode() *Node {
	return NewNode(r.ID, r.IP, r.UDPPort)
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
	log.Debug("ping received from: ", common.BytesToHex(m.ID.Bytes()))

	// response with pong
	if m.Version != VERSION {
		return
	}

	resp := &pong {
		ID: t.self.ID,
	}

	t.sendMsg(PongMsg, resp, from)
}

func (m *ping) send(t *udp, to *net.UDPAddr) {
	p := &pending {
		from: m.target,
		code: PongMsg,

		callback: func(resp interface{}, addr *net.UDPAddr) (done bool) {
			r := resp.(pong)
			n := NewNodeWithAddr(r.ID, addr)
			t.table.updateNode(n)

			return true
		},
		errorCallBack: func() { // delete this node when ping timeout, TODO add time limit
			sha := m.target.ToSha()
			t.table.deleteNode(sha)
			t.db.delete(sha)
		},
	}

	t.addpending <- p
	t.sendMsg(PingMsg, m, to)
}

func (m *findNode) handle(t *udp, from *net.UDPAddr) {
	log.Debug("find node msg")

	// response find node request
	nodes := t.table.findNodeResponse(m.target.ToSha())

	rpcs := make([]*rpcNode, len(nodes))
	for index, n := range nodes {
		rpcs[index] = &rpcNode{
			ID: n.ID,
			IP: n.IP,
			UDPPort: n.UDPPort,
		}
	}

	response := &neighbors{
		Nodes:rpcs,
		ID: t.self.ID,
	}

	t.sendMsg(NeighborsMsg, response, from)
}

func (m *findNode) send(t *udp, to *net.UDPAddr)  {
	p := &pending{
		from: m.target,
		code: NeighborsMsg,

		callback: func(resp interface{}, addr *net.UDPAddr) (done bool) {
			r := resp.(neighbors)

			if r.Nodes == nil || len(r.Nodes) == 0 {
				return true
			}

			found := false
			for _, n := range r.Nodes {
				if n.ID == m.target {
					found = true
				}

				node := n.ToNode()
				t.table.addNode(node)
				t.db.add(node.sha, node)
			}

			if !found {
				nodes := t.table.findNodeForRequest(m.target.ToSha())
				sendFindNodeRequest(t, nodes, m.target)
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
			ID: n.ID,
			target: target,
		}

		addr := &net.UDPAddr{
			IP: n.IP,
			Port: int(n.UDPPort),
		}

		f.send(u, addr)
	}
}