/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"container/list"
	"github.com/seeleteam/go-seele/crypto"
	"net"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
)

const (
	responseTimeout = 500 * time.Millisecond
)

type udp struct {
	conn  *net.UDPConn
	self  *Node
	table *Table

	localAddr *net.UDPAddr
	db *database

	gotreply   chan reply
	addpending chan *pending
	write chan *send
}

type pending struct {
	from NodeID
	code MsgType

	deadline time.Time

	callback func(resp interface{}, addr *net.UDPAddr) (done bool)

	errorCallBack func()
}

type send struct {
	buff []byte
	target *net.UDPAddr
}

type reply struct {
	from NodeID
	code MsgType

	addr *net.UDPAddr

	data interface{}
}

func NewUDP(id NodeID, addr *net.UDPAddr) *udp {
	transport := &udp{
		conn:      getUDPConn(addr),
		table:     NewTable(id, addr),
		self:      NewNodeWithAddr(id, addr),
		localAddr: addr,

		db: NewDatabase(),

		gotreply: make(chan reply, 1),
		addpending: make(chan *pending, 1),
		write: make(chan *send, 1),
	}

	return transport
}

func (u *udp) sendMsg(t MsgType, msg interface{}, target *net.UDPAddr) {
	encoding, err := common.Encoding(msg)
	if err != nil {
		log.Info(err.Error())
		return
	}

	buff := generateBuff(t, encoding)
	s := &send{
		buff:buff,
		target:target,
	}
	u.write <- s
}

func sendMsg(buff []byte, source, target *net.UDPAddr) {
	conn, err := net.DialUDP("udp", source, target)
	if err != nil {
		log.Info(err.Error())
	}
	defer conn.Close()

	//log.Debug("buff length:", len(buff))
	n, err := conn.Write(buff)
	if err != nil {
		log.Info(err.Error())
	}

	if n != len(buff) {
		log.Error("send msg failed, expected length: %d, actuall length: %d", len(buff), n)
	}
}

func (u *udp) sendLoop() {
	for {
		select {
		case s := <- u.write:
			//log.Debug("send msg to: %d", s.to.Port)
			sendMsg(s.buff, u.localAddr, s.target)
		}
	}
}

func (u *udp) handleMsg(from *net.UDPAddr, data []byte) {
	if len(data) > 0 {
		code := byteToMsgType(data[0])

		//log.Debug("msg type: %d", code)
		switch code {
		case PingMsg:
			msg := &ping{}
			err := common.Decoding(data[1:], &msg)
			if err != nil {
				log.Info(err.Error())
				return
			}

			// response ping
			msg.handle(u, from)
		case PongMsg:
			msg := &pong{}
			err := common.Decoding(data[1:], &msg)
			if err != nil {
				log.Info(err.Error())
				return
			}

			r := reply {
				from: msg.SelfID,
				code: code,
				addr:from,
				data: msg,
			}

			u.gotreply <- r
		case FindNodeMsg:
			msg := &findNode{}

			err := common.Decoding(data[1:], &msg)
			if err != nil {
				log.Info(err.Error())
				return
			}

			//response find
			msg.handle(u, from)
		case NeighborsMsg:
			msg := &neighbors{}
			err := common.Decoding(data[1:], &msg)
			if err != nil {
				log.Info(err.Error())
				return
			}

			r := reply {
				from: msg.SelfID,
				code: code,
				addr:from,
				data: msg,
			}

			u.gotreply <- r
		}
	} else {
		log.Info("wrong length")
	}
}

func (u *udp) readLoop() {
	for {
		data := make([]byte, 1024)
		n, remoteAddr, err := u.conn.ReadFromUDP(data)
		if err != nil {
			log.Info(err.Error())
		}

		//log.Info("get msg from: %d", remoteAddr.Port)

		data = data[:n]
		u.handleMsg(remoteAddr, data)
	}
}

func (u *udp) loopReply() {
	pendingList := list.New()

	var timeout *time.Timer
	defer timeout.Stop()

	resetTimer := func () {
		minTime := responseTimeout
		now := time.Now()
		for el := pendingList.Front(); el != nil; el = el.Next() {
			p := el.Value.(*pending)
			duration := p.deadline.Sub(now)
			if duration < 0 {
				pendingList.Remove(el)
			} else {
				if duration < minTime {
					minTime = duration
				}
			}
		}

		timeout = time.NewTimer(minTime)
	}

	resetTimer()

	for {
		select {
		case r := <- u.gotreply:
			//log.Debug("reply from code %d, %d", r.code, common.BytesToHex(r.from.Bytes()))
			for el := pendingList.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)

				//log.Debug("pending %d %d", p.code, common.BytesToHex(p.from.Bytes()))

				if p.from == r.from && p.code == r.code {
					if p.callback(r.data, r.addr) {
						pendingList.Remove(el)
						break
					}
				}
			}
		case p := <- u.addpending:
			p.deadline = time.Now().Add(responseTimeout)
			pendingList.PushBack(p)
		case <- timeout.C:
			for el := pendingList.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)
				if p.deadline.Sub(time.Now()) <= 0 {
					p.errorCallBack()
					pendingList.Remove(el)
				}
			}

			resetTimer()
		}
	}
}

func getRandomNodeID() NodeID {
	keypair, err := crypto.GenerateKey()
	if err != nil {
		log.Info(err.Error())
	}

	buff := crypto.FromECDSAPub(&keypair.PublicKey)

	id, err := BytesTOID(buff[1:])
	if err != nil {
		log.Fatal(err.Error())
	}

	return id
}


func (u *udp) discovery() {
	for {
		id := getRandomNodeID()

		nodes := u.table.findNodeForRequest(id.ToSha())
		sendFindNodeRequest(u, nodes, id)

		time.Sleep(DISCOVERYINTERVER)
	}
}

func (u *udp) pingPongService()  {
	for {
		copyMap := u.db.getCopy()

		for _, value := range copyMap {
			p := &ping{
				Version: VERSION,
				SelfID:  u.self.ID,

				to: value.ID,
			}

			addr := &net.UDPAddr{
				IP: value.IP,
				Port: int(value.UDPPort),
			}

			p.send(u, addr)

			time.Sleep(PINGPONGINTERVER)
		}
	}
}

func (u *udp) StartServe() {
	go u.readLoop()
	go u.loopReply()
	go u.discovery()
	go u.pingPongService()
	go u.sendLoop()
}