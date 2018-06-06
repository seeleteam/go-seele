/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"container/list"
	"fmt"
	rand2 "math/rand"
	"net"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

const (
	responseTimeout = 15 * time.Second

	pingpongInterval      = 20 * time.Second // sleep between ping pong, must big than response time out
	discoveryInterval     = 25 * time.Second // sleep between discovery, must big than response time out
	addTrustNodesInterval = 30 * time.Second // sleep between add trustNodes, must big than response time out
)

type udp struct {
	conn       *net.UDPConn
	self       *Node
	table      *Table
	trustNodes []*Node

	db        *Database
	localAddr *net.UDPAddr

	gotReply   chan *reply
	addPending chan *pending
	writer     chan *send

	log *log.SeeleLog
}

type pending struct {
	from *Node
	code msgType

	deadline time.Time

	callback func(resp interface{}, addr *net.UDPAddr) (done bool)

	errorCallBack func()
}

type send struct {
	toId   common.Address
	toAddr *net.UDPAddr
	buff   []byte
	//to   *Node
	code msgType
}

type reply struct {
	fromId   common.Address
	fromAddr *net.UDPAddr
	//from *Node
	code msgType

	err bool // got error when send msg

	data interface{}
}

func newUDP(id common.Address, addr *net.UDPAddr, shard uint) *udp {
	log := log.GetLogger("discovery", common.LogConfig.PrintLog)
	conn, err := getUDPConn(addr)
	if err != nil {
		panic(fmt.Sprintf("listen addr %s failed", addr.String()))
	}

	transport := &udp{
		conn:      conn,
		table:     newTable(id, addr, shard, log),
		self:      NewNodeWithAddr(id, addr, shard),
		localAddr: addr,

		db: NewDatabase(log),

		gotReply:   make(chan *reply, 1),
		addPending: make(chan *pending, 1),
		writer:     make(chan *send, 1),

		log: log,
	}

	return transport
}

func (u *udp) sendMsg(t msgType, msg interface{}, toId common.Address, toAddr *net.UDPAddr) {
	encoding, err := common.Serialize(msg)
	if err != nil {
		u.log.Info(err.Error())
		return
	}

	buff := generateBuff(t, encoding)
	s := &send{
		buff:   buff,
		toId:   toId,
		toAddr: toAddr,
		//to:   to,
		code: t,
	}
	u.writer <- s
}

func (u *udp) sendConnMsg(buff []byte, conn *net.UDPConn, to *net.UDPAddr) bool {
	//log.Debug("buff length:", len(buff))
	n, err := conn.WriteToUDP(buff, to)
	if err != nil {
		u.log.Info("send msg failed:%s", err.Error())
		return false
	}

	if n != len(buff) {
		u.log.Error("send msg failed, expected length: %d, actuall length: %d", len(buff), n)
		return false
	}

	return true
}

func (u *udp) sendLoop() {
	for {
		select {
		case s := <-u.writer:
			//log.Debug("send msg to: %d", s.to.Port)
			success := u.sendConnMsg(s.buff, u.conn, s.toAddr)
			if !success {
				r := &reply{
					fromId:   s.toId,
					fromAddr: s.toAddr,
					//from: s.to,
					code: s.code,
					err:  true,
				}

				u.gotReply <- r
			}
		}
	}
}

func (u *udp) handleMsg(from *net.UDPAddr, data []byte) {
	if len(data) > 0 {
		code := byteToMsgType(data[0])

		u.log.Debug("receive msg type: %s", codeToStr(code))
		switch code {
		case pingMsgType:
			msg := &ping{}
			err := common.Deserialize(data[1:], &msg)
			if err != nil {
				u.log.Warn(err.Error())
				return
			}

			// response ping
			msg.handle(u, from)
		case pongMsgType:
			msg := &pong{}
			err := common.Deserialize(data[1:], &msg)
			if err != nil {
				u.log.Warn(err.Error())
				return
			}

			r := &reply{
				fromId:   msg.SelfID,
				fromAddr: from,
				code:     code,
				data:     msg,
				err:      false,
			}

			u.gotReply <- r
		case findNodeMsgType:
			msg := &findNode{}

			err := common.Deserialize(data[1:], &msg)
			if err != nil {
				u.log.Warn(err.Error())
				return
			}

			//response find
			msg.handle(u, from)
		case neighborsMsgType:
			msg := &neighbors{}
			err := common.Deserialize(data[1:], &msg)
			if err != nil {
				u.log.Warn(err.Error())
				return
			}

			r := &reply{
				fromId:   msg.SelfID,
				fromAddr: from,
				code:     code,
				data:     msg,
				err:      false,
			}

			u.gotReply <- r
		case findShardNodeMsgType:
			msg := &findShardNode{}
			err := common.Deserialize(data[1:], &msg)
			if err != nil {
				u.log.Warn(err.Error())
				return
			}

			msg.handle(u, from)
		case shardNodeMsgType:
			msg := &shardNode{}
			err := common.Deserialize(data[1:], &msg)
			if err != nil {
				u.log.Warn(err.Error())
				return
			}

			r := &reply{
				fromId:   msg.SelfID,
				fromAddr: from,
				code:     code,
				data:     msg,
				err:      false,
			}

			u.gotReply <- r
		default:
			u.log.Error("unknown code %d", code)
		}
	} else {
		u.log.Info("wrong length")
	}
}

func (u *udp) readLoop() {
	for {
		data := make([]byte, 1024)
		n, remoteAddr, err := u.conn.ReadFromUDP(data)
		if err != nil {
			u.log.Info(err.Error())
		}

		data = data[:n]
		u.handleMsg(remoteAddr, data)
	}
}

func (u *udp) loopReply() {
	pendingList := list.New()

	var timeout = time.NewTimer(0)
	<-timeout.C
	defer timeout.Stop()

	resetTimer := func() {
		minTime := responseTimeout
		now := time.Now()
		for el := pendingList.Front(); el != nil; el = el.Next() {
			p := el.Value.(*pending)
			duration := p.deadline.Sub(now)
			if duration < 0 {
			} else {
				if duration < minTime {
					minTime = duration
				}
			}
		}

		// if there is no pending request, stop timer
		if pendingList.Len() == 0 {
			timeout.Stop()
		} else {
			timeout.Reset(minTime)
		}
	}

	for {
		resetTimer()

		select {
		case r := <-u.gotReply:
			for el := pendingList.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)

				if p.from.ID == r.fromId && p.code == r.code {
					if r.err {
						p.errorCallBack()
						pendingList.Remove(el)
					} else {
						if p.callback(r.data, r.fromAddr) {
							pendingList.Remove(el)
						}
					}

					break
				}
			}
		case p := <-u.addPending:
			p.deadline = time.Now().Add(responseTimeout)
			pendingList.PushBack(p)
		case <-timeout.C:
			for el := pendingList.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)
				if p.deadline.Sub(time.Now()) <= 0 {
					u.log.Info("time out to wait for msg with msg type %s", codeToStr(p.code))
					p.errorCallBack()
					pendingList.Remove(el)
				}
			}

			resetTimer()
		}
	}
}

func (u *udp) discovery(isFast bool) {
	for {
		id, err := crypto.GenerateRandomAddress()
		if err != nil {
			u.log.Error(err.Error())
			continue
		}

		nodes := u.table.findNodeForRequest(crypto.HashBytes(id.Bytes()))

		u.log.Debug("query node with id: %s", id.ToHex())
		sendFindNodeRequest(u, nodes, *id)

		if !isFast {
			time.Sleep(discoveryInterval)
		}

		for i := 1; i < common.ShardNumber+1; i++ {
			shardBucket := u.table.shardBuckets[i]
			size := shardBucket.size()
			if size < bucketSize {
				var node *Node
				if size == 0 {
					node = u.db.getRandNode()
				} else {
					// request same shard node will find more nodes
					selectNode := rand2.Intn(size)
					node = shardBucket.get(selectNode)
				}

				if node == nil {
					continue
				}

				sendFindShardNodeRequest(u, uint(i), node)

				if !isFast {
					time.Sleep(discoveryInterval)
				}
			}
		}

		if isFast {
			time.Sleep(discoveryInterval)
		}

		if isFast {
			enough := true
			for i := 1; i < common.ShardNumber+1; i++ {
				if uint(i) == u.self.Shard {
					continue
				}

				if u.table.shardBuckets[i].size() < shardTargeNodeNumber {
					enough = false
				}
			}

			// if we get enough peers, stop fast discovery
			if enough {
				break
			}
		}
	}
}

func (u *udp) discoveryWithTwoStags() {
	// discovery with two stage
	// 1. fast discovery, with small network interval. fast stage will stop when got minimal number of peers
	// 2. normal discovery, with normal network interval
	//u.discovery(true) // disable fast discovery

	u.discovery(false)
}

func (u *udp) loopAddTrustNodes() {
	for {
		u.addTrustNodes()
		time.Sleep(addTrustNodesInterval)
	}
}

func (u *udp) addTrustNodes() {
	for i := range u.trustNodes {
		if _, ok := u.db.FindByNodeID(u.trustNodes[i].ID); !ok {
			u.addNode(u.trustNodes[i])
		}
	}
}

func (u *udp) pingPongService() {
	for {
		copyMap := u.db.GetCopy()

		for _, value := range copyMap {
			p := &ping{
				Version:   discoveryProtocolVersion,
				SelfID:    u.self.ID,
				SelfShard: u.self.Shard,

				to: value,
			}

			p.send(u)
			time.Sleep(pingpongInterval)
		}
	}
}

func (u *udp) StartServe(nodeDir string) {
	go u.readLoop()
	go u.loopReply()
	go u.discoveryWithTwoStags()
	go u.pingPongService()
	go u.sendLoop()
	go u.db.StartSaveNodes(nodeDir, make(chan struct{}))
	go u.loopAddTrustNodes()
}

func (u *udp) addNode(n *Node) {
	if n == nil || u.self.ID.Equal(n.ID) {
		return
	}

	u.table.addNode(n)
	u.db.add(n)
	u.log.Info("after add node, total nodes:%d", u.db.size())
}

func (u *udp) deleteNode(n *Node) {
	selfSha := u.self.getSha()
	sha := n.getSha()
	if sha == selfSha {
		return
	}

	u.table.deleteNode(n)
	u.db.delete(sha)
	u.log.Info("after delete node, total nodes:%d", u.db.size())
}
