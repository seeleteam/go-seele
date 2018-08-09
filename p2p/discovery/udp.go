/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io/ioutil"
	rand2 "math/rand"
	"net"
	"path/filepath"
	"time"

	"github.com/orcaman/concurrent-map"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

const (
	responseTimeout = 20 * time.Second

	pingpongConcurrentNumber = 5
	pingpongInterval         = 30 * time.Second // sleep between ping pong, must big than response time out

	discoveryConcurrentNumber = 5
	discoveryInterval         = 35 * time.Second // sleep between discovery, must big than response time out

	// a node will be delete after n continuous time out.
	timeoutCountForDeleteNode = 8
)

type udp struct {
	conn           *net.UDPConn
	self           *Node
	table          *Table
	trustNodes     []*Node
	bootstrapNodes []*Node

	db        *Database
	localAddr *net.UDPAddr

	gotReply   chan *reply
	addPending chan *pending
	writer     chan *send

	log *log.SeeleLog

	timeoutNodesCount cmap.ConcurrentMap //node id -> count
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
	code   msgType
}

type reply struct {
	fromId   common.Address
	fromAddr *net.UDPAddr
	code     msgType

	err bool // got error when send msg

	data interface{}
}

func newUDP(id common.Address, addr *net.UDPAddr, shard uint) *udp {
	log := log.GetLogger("discovery")
	conn, err := getUDPConn(addr)
	if err != nil {
		panic(fmt.Sprintf("failed to listen addr %s ", addr.String()))
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

		log:               log,
		timeoutNodesCount: cmap.New(),
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
		code:   t,
	}

	u.writer <- s
}

func (u *udp) sendConnMsg(buff []byte, conn *net.UDPConn, to *net.UDPAddr) bool {
	n, err := conn.WriteToUDP(buff, to)
	if err != nil {
		u.log.Warn("failed to discover send msg to %s:%s", to, err)
		return false
	}

	if n != len(buff) {
		u.log.Warn("failed to discover sending msg to %s, expected length: %d, actual length: %d", to, len(buff), n)
		return false
	}

	return true
}

func (u *udp) sendLoop() {
	for {
		select {
		case s := <-u.writer:
			success := u.sendConnMsg(s.buff, u.conn, s.toAddr)
			if !success {
				r := &reply{
					fromId:   s.toId,
					fromAddr: s.toAddr,
					code:     s.code,
					err:      true,
				}

				u.gotReply <- r
			}
		}
	}
}

func (u *udp) handleMsg(from *net.UDPAddr, data []byte) {
	if len(data) > 0 {
		code := byteToMsgType(data[0])

		if common.PrintExplosionLog {
			u.log.Debug("receive msg type: %s", codeToStr(code))
		}
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
		// 1472 is udp max transfer size for once
		data := make([]byte, 1472)
		n, remoteAddr, err := u.conn.ReadFromUDP(data)
		if err != nil {
			u.log.Warn("failed to discover reading from udp %s", err)
			continue
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

		timeout.Reset(minTime)
	}

	resetTimer()

	for {
		select {
		case r := <-u.gotReply:
			for el := pendingList.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)

				if p.code == r.code && p.from.GetUDPAddr().String() == r.fromAddr.String() {
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
					errorMsg := fmt.Sprintf("time out to wait for msg with msg type %s for node %s", codeToStr(p.code), p.from)
					if p.code == pongMsgType {
						u.log.Info(errorMsg)
					} else {
						u.log.Debug(errorMsg)
					}

					p.errorCallBack()
					pendingList.Remove(el)
				}
			}

			resetTimer()
		}
	}
}

func (u *udp) discovery() {
	for {
		id, err := crypto.GenerateRandomAddress()
		if err != nil {
			u.log.Error(err.Error())
			continue
		}

		nodes := u.table.findNodeForRequest(crypto.HashBytes(id.Bytes()))

		u.log.Debug("query node with id: %s", id.ToHex())
		sendFindNodeRequest(u, nodes, *id)

		concurrentCount := 0
		for i := 1; i < common.ShardCount+1; i++ {
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

				concurrentCount++
				if concurrentCount == discoveryConcurrentNumber {
					time.Sleep(discoveryInterval)
					concurrentCount = 0
				}
			}
		}

		time.Sleep(discoveryInterval)
	}
}

func (u *udp) pingPongService() {
	for {
		copyMap := u.db.GetCopy()
		loopPingPongNodes := make(map[string]*Node, 0)

		// loopPingPongNodes add backup nodes, only ping pong once
		if len(u.bootstrapNodes) > 0 {
			for i := range u.bootstrapNodes {
				loopPingPongNodes[u.bootstrapNodes[i].GetUDPAddr().String()] = u.bootstrapNodes[i]
			}
			u.bootstrapNodes = nil
		}

		// loopPingPongNodes add trust nodes, loop ping pong; if bootstrapNodes have the same key, will use the trust node to update it
		if len(u.trustNodes) > 0 {
			for i := range u.trustNodes {
				loopPingPongNodes[u.trustNodes[i].GetUDPAddr().String()] = u.trustNodes[i]
			}
		}

		// loopPingPongNodes add db nodes, loop ping pong; if bootstrapNodes or trustNodes have the same key, will use the db node to update it
		if len(copyMap) > 0 {
			for _, value := range copyMap {
				loopPingPongNodes[value.GetUDPAddr().String()] = value
			}
		}

		u.log.Debug("loop ping pong nodes %d", len(loopPingPongNodes))
		concurrentCount := 0
		for _, n := range loopPingPongNodes {
			u.ping(n)

			concurrentCount++
			if concurrentCount == pingpongConcurrentNumber {
				time.Sleep(pingpongInterval)
				concurrentCount = 0
			}
		}

		time.Sleep(pingpongInterval)
	}
}

func (u *udp) ping(value *Node) {
	p := &ping{
		Version:   discoveryProtocolVersion,
		SelfID:    u.self.ID,
		SelfShard: u.self.Shard,

		to: value,
	}

	p.send(u)
}

func (u *udp) StartServe(nodeDir string) {
	go u.readLoop()
	go u.loopReply()
	go u.discovery()
	go u.pingPongService()
	go u.sendLoop()
	go u.db.StartSaveNodes(nodeDir, make(chan struct{}))
}

// only notify connect when got pong msg
func (u *udp) addNode(n *Node, notifyConnect bool) {
	if n == nil || u.self.ID.Equal(n.ID) {
		return
	}

	count := u.db.size()
	u.table.addNode(n)
	u.db.add(n, notifyConnect)

	newCount := u.db.size()
	if count != newCount {
		u.log.Info("add node %s, total nodes:%d", n, newCount)
	} else {
		u.log.Debug("got add node event, but it is already exist. total nodes didn't change:%d", newCount)
	}
}

func (u *udp) deleteNode(n *Node) {
	selfSha := u.self.getSha()
	sha := n.getSha()
	if sha == selfSha {
		return
	}

	if _, ok := u.db.FindByNodeID(n.ID); !ok {
		return
	}

	idStr := n.ID.ToHex()
	var count = 0
	value, ok := u.timeoutNodesCount.Get(idStr)
	if ok {
		count = value.(int)
	}

	count++
	if count == timeoutCountForDeleteNode {
		u.table.deleteNode(n)
		u.db.delete(sha)

		u.log.Info("after delete node %s, total nodes:%d", n, u.db.size())
		u.timeoutNodesCount.Remove(idStr)
	} else {
		u.log.Info("node %s got time out, current timeout count %d", n, count)
		u.timeoutNodesCount.Set(idStr, count)
	}
}

func (u *udp) loadNodes(nodeDir string) {
	fileFullPath := filepath.Join(nodeDir, NodesBackupFileName)

	if !common.FileOrFolderExists(fileFullPath) {
		u.log.Debug("nodes info backup file isn't exists in the path:%s", fileFullPath)
		return
	}

	data, err := ioutil.ReadFile(fileFullPath)
	if err != nil {
		u.log.Error("failed to read nodes info backup file for:[%s]", err)
		return
	}

	var nodes []string
	err = json.Unmarshal(data, &nodes)
	if err != nil {
		u.log.Error("failed to unmarshal nodes for:[%s]", err)
		return
	}

	for i := range nodes {
		n, err := NewNodeFromString(nodes[i])
		if err != nil {
			u.log.Error("new node from string failed for:[%s]", err)
			continue
		}
		u.bootstrapNodes = append(u.bootstrapNodes, n)
	}

	u.log.Debug("load %d nodes from back file", len(u.bootstrapNodes))
}
