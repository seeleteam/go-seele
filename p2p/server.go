/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	//"crypto/ecdsa"

	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/aristanetworks/goarista/monotime"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p/discovery"
	//"github.com/ethereum/go-ethereum/p2p/discover"
)

const (
	// Maximum number of concurrently handshaking inbound connections.
	maxAcceptConns = 50

	defaultDialTimeout = 15 * time.Second

	// Maximum time allowed for reading a complete message.
	frameReadTimeout = 30 * time.Second

	// Maximum amount of time allowed for writing a complete message.
	frameWriteTimeout = 20 * time.Second

	inboundConn  = 1
	outboundConn = 2
)

// Config holds Server options.
type Config struct {
	// Use common.MakeName to create a name that follows existing conventions.
	Name string //`toml:"-"`

	// MaxPendingPeers is the maximum number of peers that can be pending in the
	// handshake phase, counted separately for inbound and outbound connections.
	// Zero defaults to preset values.
	MaxPendingPeers int `toml:",omitempty"`

	MyNodeID string
	// pre-configured nodes.
	StaticNodes []*discovery.Node

	KadPort string // udp port for Kad network

	// Protocols should contain the protocols supported by the server.
	Protocols []ProtocolInterface `toml:"-"`

	// p2p.server will listen for incoming tcp connections.
	ListenAddr string
}

// Server manages all p2p peer connections.
type Server struct {
	// Config fields may not be modified while the server is running.
	Config

	lock    sync.Mutex // protects running
	running bool

	kadDB    *discovery.Database
	listener net.Listener

	quit chan struct{}

	addpeer chan *Peer
	delpeer chan *Peer
	loopWG  sync.WaitGroup // loop, listenLoop

	peers map[common.Address]*Peer
	log   *log.SeeleLog
}

// Start starts running the server.
func (srv *Server) Start() (err error) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.running {
		return errors.New("server already running")
	}
	srv.log = log.GetLogger("p2p", true)
	if srv.log == nil {
		return errors.New("p2p Create logger error")
	}
	srv.running = true
	srv.peers = make(map[common.Address]*Peer)

	srv.log.Info("Starting P2P networking...")
	srv.quit = make(chan struct{})
	srv.addpeer = make(chan *Peer)
	srv.delpeer = make(chan *Peer)

	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%s", srv.KadPort))
	if err != nil {
		return err
	}

	myId := common.HexToAddress(srv.MyNodeID)
	srv.kadDB = discovery.StartService(myId, addr, srv.StaticNodes)
	if err := srv.startListening(); err != nil {
		return err
	}

	for _, proto := range srv.Protocols {
		go func() {
			srv.loopWG.Add(1)
			proto.Run()
			close(proto.GetBaseProtocol().AddPeerCh)
			close(proto.GetBaseProtocol().DelPeerCh)
			close(proto.GetBaseProtocol().ReadMsgCh)
			srv.loopWG.Done()
		}()
	}
	srv.loopWG.Add(1)
	go srv.run()
	srv.running = true

	return nil
}

func (srv *Server) run() {
	defer srv.loopWG.Done()
	peers := srv.peers
	srv.log.Info("p2p start running...")
	checkTimer := time.NewTimer(10 * time.Second)
running:
	for {
		//srv.scheduleTasks()
		select {
		case <-checkTimer.C:
			checkTimer.Reset(5 * time.Second)
			srv.scheduleTasks()
		case <-srv.quit:
			// The server was stopped. Run the cleanup logic.
			break running
		case c := <-srv.addpeer:
			_, ok := peers[c.node.ID]

			if ok {
				// node already connected, need close this connection
				srv.log.Info("server.run  <-srv.addpeer, len(peers)=%d. nodeid already connected", len(peers))
				c.Disconnect(discAlreadyConnected)
			} else {
				peers[c.node.ID] = c
				srv.log.Info("server.run  <-srv.addpeer, len(peers)=%d, len(srv.peers)=%d", len(peers), len(srv.peers))
			}
		case pd := <-srv.delpeer:
			curPeer, ok := peers[pd.node.ID]
			if ok && curPeer == pd {
				delete(peers, pd.node.ID)
				srv.log.Info("server.run delpeer recved. peer match. remove peer. peers num=%d", len(peers))
			} else {
				srv.log.Info("server.run delpeer recved. peer not match")
			}
		}
	}

	// Disconnect all peers.
	for _, p := range peers {
		p.Disconnect(discServerQuit)
	}

	for len(peers) > 0 {
		p := <-srv.delpeer
		delete(peers, p.node.ID)
	}
}

//scheduleTasks
func (srv *Server) scheduleTasks() {
	// TODO select nodes from ntab to connect
	nodeMap := srv.kadDB.GetCopy()
	srv.log.Info("scheduleTasks called... nodeMap=[%d] srv.peers=%d", len(nodeMap), len(srv.peers))
	for _, node := range nodeMap {
		_, ok := srv.peers[node.ID]
		if ok {
			continue
		}
		//TODO UDPPort==> TCPPort
		addr, _ := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", node.IP.String(), node.UDPPort))
		conn, err := net.DialTimeout("tcp", addr.String(), defaultDialTimeout)
		srv.log.Info("scheduleTasks connecting... %s", addr.String())
		if err != nil {
			if conn != nil {
				conn.Close()
			}
			continue
		}
		go srv.setupConn(conn, outboundConn, node)
	}
	/*for _, node := range srv.StaticNodes {
		_, ok := srv.peers[node.ID]
		if ok {
			continue
		}
		//TODO UDPPort==> TCPPort
		addr, _ := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", node.IP.String(), node.UDPPort))
		conn, err := net.DialTimeout("tcp", addr.String(), defaultDialTimeout)
		if err != nil {
			if conn != nil {
				conn.Close()
			}
			continue
		}
		go srv.setupConn(conn, outboundConn, node)
	}*/
}

func (srv *Server) startListening() error {
	// Launch the TCP listener.
	listener, err := net.Listen("tcp", srv.ListenAddr)
	if err != nil {
		return err
	}
	laddr := listener.Addr().(*net.TCPAddr)
	srv.ListenAddr = laddr.String()
	srv.listener = listener
	srv.loopWG.Add(1)
	go srv.listenLoop()
	return nil
}

type tempError interface {
	Temporary() bool
}

// listenLoop runs in its own goroutine and accepts inbound connections.
func (srv *Server) listenLoop() {
	defer srv.loopWG.Done()
	// If all slots are taken, no further connections are accepted.
	tokens := maxAcceptConns
	if srv.MaxPendingPeers > 0 {
		tokens = srv.MaxPendingPeers
	}
	slots := make(chan struct{}, tokens)
	for i := 0; i < tokens; i++ {
		slots <- struct{}{}
	}

	for {
		// Wait for a handshake slot before accepting.
		<-slots
		var (
			fd  net.Conn
			err error
		)
		for {
			fd, err = srv.listener.Accept()
			if tempErr, ok := err.(tempError); ok && tempErr.Temporary() {
				continue
			} else if err != nil {
				srv.log.Error("p2p.listenLoop accept err. %s", err)
				return
			}
			break
		}
		go func() {
			srv.setupConn(fd, inboundConn, nil)
			slots <- struct{}{}
		}()
	}
}

// setupConn TODO add encypt-handshake.
func (srv *Server) setupConn(fd net.Conn, flags int, dialDest *discovery.Node) error {
	peer := &Peer{
		conn:     &connection{fd: fd},
		created:  monotime.Now(),
		disc:     make(chan uint),
		closed:   make(chan struct{}),
		protoMap: make(map[uint16]*Protocol),
		capMap:   make(map[string]uint16),
		log:      srv.log,
		node:     dialDest,
	}

	var caps []Cap
	for _, proto := range srv.Protocols {
		caps = append(caps, proto.GetBaseProtocol().cap())
	}
	wrapMsg := &msg{
		protoCode: ctlProtoCode,
		msgCode:   ctlMsgProtoHandshake,
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	myNounce := r.Uint32()
	handshakeMsg := &ProtoHandShake{Caps: caps, Nounce: myNounce}
	nodeID := common.HexToAddress(srv.MyNodeID)
	copy(handshakeMsg.NodeID[0:], nodeID[0:])

	// Serialize should handle big- little- endian?
	buffer, err := common.Serialize(handshakeMsg)
	if err != nil {
		peer.close()
		return err
	}
	wrapMsg.payload = make([]byte, len(buffer))
	copy(wrapMsg.payload, buffer)
	wrapMsg.size = uint32(len(wrapMsg.payload))
	peer.sendRawMsg(wrapMsg)

	recvWrapMsg, err := peer.recvRawMsg()
	if err != nil {
		peer.close()
		return err
	}

	recvMsg := &ProtoHandShake{}
	if err := common.Deserialize(recvWrapMsg.payload, recvMsg); err != nil {
		peer.close()
		return err
	}

	peerCaps, peerNodeID, peerNounce := recvMsg.Caps, recvMsg.NodeID, recvMsg.Nounce
	// TODO need merge caps and order by cap name, make sure having the same order at each end
	// TODO compute a secret key by myNounce and peerNounce
	protoCode := uint16(baseProtoCode)
	for _, proto := range srv.Protocols {
		peer.protoMap[protoCode] = proto.GetBaseProtocol()
		peer.capMap[proto.GetBaseProtocol().cap().String()] = protoCode
		protoCode++
	}

	if flags == inboundConn {
		var peerNode *discovery.Node
		nodeMap := srv.kadDB.GetCopy()
		for _, node := range nodeMap {
			if bytes.Equal(node.ID[0:], peerNodeID[0:]) {
				peerNode = node
				break
			}
		}
		if peerNode == nil {
			srv.log.Info("p2p.setupConn conn handshaked, not found nodeID")
			peer.close()
			return errors.New("not found nodeID in discovery database!")
		}
		peer.node = peerNode
	}

	srv.log.Info("p2p.setupConn conn handshaked.  peerNounce=%d peerCaps=%s", int(peerNounce), peerCaps)
	go func() {
		srv.loopWG.Add(1)
		srv.addpeer <- peer
		peer.run()
		srv.delpeer <- peer
		srv.loopWG.Done()
	}()
	return nil
}
