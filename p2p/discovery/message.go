/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

type msgType uint8

const (
	// add msg type flag at the first byte of the message
	pingMsgType          msgType = 1
	pongMsgType          msgType = 2
	findNodeMsgType      msgType = 3
	neighborsMsgType     msgType = 4
	findShardNodeMsgType msgType = 5
	shardNodeMsgType     msgType = 6
)

func codeToStr(code msgType) string {
	switch code {
	case pingMsgType:
		return "pingMsgType"
	case pongMsgType:
		return "pongMsgType"
	case findNodeMsgType:
		return "findNodeMsgType"
	case neighborsMsgType:
		return "neighborsMsgType"
	case findShardNodeMsgType:
		return "findShardNodeMsgType"
	case shardNodeMsgType:
		return "shardNodeMsgType"
	default:
		return "unkwown"
	}
}

const (
	discoveryProtocolVersion uint = 1
)

type ping struct {
	Version   uint // TODO add version check
	SelfID    common.Address
	SelfShard uint

	to *Node
}

type pong struct {
	SelfID    common.Address
	SelfShard uint
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
	SelfID       common.Address
	RequestShard uint // request shard info

	to *Node
}

type shardNode struct {
	SelfID       common.Address
	RequestShard uint // request shard info
	Nodes        []*rpcNode
}

type rpcNode struct {
	SelfID  common.Address
	IP      net.IP
	UDPPort uint16
	Shard   uint
}

func (r *rpcNode) ToNode() *Node {
	return NewNode(r.SelfID, r.IP, int(r.UDPPort), r.Shard)
}

func ConvertToRpcNode(n *Node) *rpcNode {
	return &rpcNode{
		SelfID:  n.ID,
		IP:      n.IP,
		UDPPort: uint16(n.UDPPort),
		Shard:   n.Shard,
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
	node := NewNodeWithAddr(m.SelfID, from, m.SelfShard)
	t.addNode(node, false)
	t.timeoutNodesCount.Set(m.SelfID.ToHex(), 0)

	// response with pong
	if m.Version != discoveryProtocolVersion {
		return
	}

	resp := &pong{
		SelfID:    t.self.ID,
		SelfShard: t.self.Shard,
	}

	t.log.Debug("received [pingMsg] and send [pongMsg] to: %s", node)
	t.sendMsg(pongMsgType, resp, node.ID, node.GetUDPAddr())
}

// send send ping message and handle callback
func (m *ping) send(t *udp) {
	t.log.Debug("send [pingMsg] to: %s", m.to)

	p := &pending{
		from: m.to,
		code: pongMsgType,

		callback: func(resp interface{}, addr *net.UDPAddr) (done bool) {
			r := resp.(*pong)
			n := NewNodeWithAddr(r.SelfID, addr, r.SelfShard)
			t.addNode(n, true)
			t.timeoutNodesCount.Set(n.ID.ToHex(), 0)

			t.log.Debug("received [pongMsg] from: %s", n)

			return true
		},
		errorCallBack: func() { // delete this node when ping timeout, TODO add time limit
			t.deleteNode(m.to)
		},
	}

	t.addPending <- p
	t.sendMsg(pingMsgType, m, m.to.ID, m.to.GetUDPAddr())
}

// handle response find node request
func (m *findNode) handle(t *udp, from *net.UDPAddr) {
	t.log.Debug("received request [findNodeMsg] from: %s, id: %s", from, m.SelfID.ToHex())

	nodes := t.table.findNodeWithTarget(crypto.HashBytes(m.QueryID.Bytes()))

	if len(nodes) == 0 {
		return
	}

	rpcs := make([]*rpcNode, len(nodes))
	for index, n := range nodes {
		rpcs[index] = ConvertToRpcNode(n)
	}

	response := &neighbors{
		Nodes:  rpcs,
		SelfID: t.self.ID,
	}

	t.sendMsg(neighborsMsgType, response, m.SelfID, from)
}

// send send find node message and handle callback
func (m *findNode) send(t *udp) {
	t.log.Debug("send request [findNodeMsg] to: %s", m.to)

	p := &pending{
		from: m.to,
		code: neighborsMsgType,

		callback: func(resp interface{}, addr *net.UDPAddr) (done bool) {
			r := resp.(*neighbors)

			t.log.Debug("received [neighborsMsg] from: %s with %d nodes", r.SelfID.ToHex(), len(r.Nodes))
			if r.Nodes == nil || len(r.Nodes) == 0 {
				return true
			}

			for _, n := range r.Nodes {
				t.log.Debug("received node: %s", n.SelfID.ToHex())

				node := n.ToNode()
				t.addNode(node, false)
			}

			return true
		},
		errorCallBack: func() {
			// do nothing
		},
	}

	t.addPending <- p
	t.sendMsg(findNodeMsgType, m, m.to.ID, m.to.GetUDPAddr())
}

func sendFindNodeRequest(u *udp, nodes []*Node, target common.Address) {
	if nodes == nil || len(nodes) == 0 {
		return
	}

	concurrentCount := 0
	for _, n := range nodes {
		f := &findNode{
			SelfID:  u.self.ID,
			QueryID: target,
			to:      n,
		}

		f.send(u)

		concurrentCount++
		if concurrentCount == discoveryConcurrentNumber {
			time.Sleep(discoveryInterval)
			concurrentCount = 0
		}
	}

	time.Sleep(discoveryInterval)
}

func sendFindShardNodeRequest(u *udp, shard uint, to *Node) {
	query := &findShardNode{
		SelfID:       u.self.ID,
		RequestShard: shard,

		to: to,
	}

	query.send(u)
}

func (m *findShardNode) send(t *udp) {
	t.log.Debug("send request [findShardNodeMsg], shard: %d, to node: %s", m.RequestShard, m.to)

	p := &pending{
		from: m.to,
		code: shardNodeMsgType,

		callback: func(resp interface{}, addr *net.UDPAddr) (done bool) {
			r := resp.(*shardNode)
			t.log.Debug("got response [shardNodeMsg] with nodes number %d in shard %d from:%s",
				len(r.Nodes), r.RequestShard, addr)
			for _, node := range r.Nodes {
				t.addNode(node.ToNode(), false)
			}

			return true
		},
		errorCallBack: func() {
		},
	}

	t.addPending <- p
	t.sendMsg(findShardNodeMsgType, m, m.to.ID, m.to.GetUDPAddr())
}

func (m *findShardNode) handle(t *udp, from *net.UDPAddr) {
	t.log.Debug("got request [findShardNodeMsg] from: %s, find shard %d", from, m.RequestShard)

	var nodes []*Node
	if m.RequestShard == t.self.Shard {
		nodes = t.table.GetRandNodes(responseNodeNumber)
	} else {
		bucket := t.table.shardBuckets[m.RequestShard]
		nodes = bucket.getRandNodes(responseNodeNumber)
	}

	if len(nodes) == 0 {
		return
	}

	rpcNodes := make([]*rpcNode, len(nodes))
	for i := 0; i < len(nodes); i++ {
		rpcNodes[i] = ConvertToRpcNode(nodes[i])
	}

	response := &shardNode{
		SelfID:       t.self.ID,
		RequestShard: m.RequestShard,
		Nodes:        rpcNodes,
	}

	t.sendMsg(shardNodeMsgType, response, m.SelfID, from)
}
