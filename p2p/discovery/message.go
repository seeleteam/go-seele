/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"net"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/common/hexutil"
)

type msgType uint8

const (
	// add msg type flag at the first byte of the message
	pingMsgType      msgType = 1
	pongMsgType      msgType = 2
	findNodeMsgType  msgType = 3
	neighborsMsgType msgType = 5
)

const (
	discoveryProtocolVersion uint = 1
)

type ping struct {
	Version uint // TODO add version check
	SelfID  NodeID

	to *Node
}

type pong struct {
	SelfID NodeID
}

type findNode struct {
	SelfID  NodeID
	QueryID NodeID

	to *Node
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
	log.Debug("received ping from: %s", hexutil.BytesToHex(m.SelfID.Bytes()))

	// response with pong
	if m.Version != discoveryProtocolVersion {
		return
	}

	resp := &pong{
		SelfID: t.self.ID,
	}

	t.sendMsg(pongMsgType, resp, NewNodeWithAddr(m.SelfID, from))
}

func (m *ping) send(t *udp) {
	log.Debug("send ping msg to: %s", hexutil.BytesToHex(m.to.ID.Bytes()))

	p := &pending{
		from: m.to,
		code: pongMsgType,

		callback: func(resp interface{}, addr *net.UDPAddr) (done bool) {
			r := resp.(*pong)
			n := NewNodeWithAddr(r.SelfID, addr)
			t.table.updateNode(n)

			log.Debug("received pong msg: %s", hexutil.BytesToHex(r.SelfID.Bytes()))

			return true
		},
		errorCallBack: func() { // delete this node when ping timeout, TODO add time limit
			sha := m.to.ID.ToSha()
			t.deleteNode(sha)
		},
	}

	t.addPending <- p
	t.sendMsg(pingMsgType, m, m.to)
}

// handle response find node request
func (m *findNode) handle(t *udp, from *net.UDPAddr) {
	log.Debug("received find node request from: %s", hexutil.BytesToHex(m.SelfID.Bytes()))

	node := NewNodeWithAddr(m.SelfID, from)
	t.addNode(node)

	nodes := t.table.findNodeWithTarget(m.QueryID.ToSha(), m.SelfID.ToSha())

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

	t.sendMsg(neighborsMsgType, response, NewNodeWithAddr(m.SelfID, from))
}

func (m *findNode) send(t *udp) {
	log.Debug("send find msg to: %s", hexutil.BytesToHex(m.to.ID.Bytes()))

	p := &pending{
		from: m.to,
		code: neighborsMsgType,

		callback: func(resp interface{}, addr *net.UDPAddr) (done bool) {
			r := resp.(*neighbors)

			log.Debug("received neighbors msg from: %s", hexutil.BytesToHex(r.SelfID.Bytes()))

			if r.Nodes == nil || len(r.Nodes) == 0 {
				return true
			}

			found := false
			for _, n := range r.Nodes {
				if n.SelfID == m.QueryID {
					found = true
				}

				node := n.ToNode()
				t.addNode(node)
			}

			if !found {
				nodes := t.table.findNodeWithTarget(m.QueryID.ToSha(), m.SelfID.ToSha())
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

func sendFindNodeRequest(u *udp, nodes []*Node, target NodeID) {
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
