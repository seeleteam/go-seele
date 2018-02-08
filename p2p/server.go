/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	//"crypto/ecdsa"

	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/aristanetworks/goarista/monotime"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/p2p/discovery"
	//"github.com/ethereum/go-ethereum/p2p/discover"
)

const (
	defaultDialTimeout = 15 * time.Second

	// Maximum number of concurrently handshaking inbound connections.
	maxAcceptConns = 50

	// Maximum number of concurrently dialing outbound connections.
	maxActiveDialTasks = 16

	// Maximum time allowed for reading a complete message.
	// This is effectively the amount of time a connection can be idle.
	frameReadTimeout = 30 * time.Second

	// Maximum amount of time allowed for writing a complete message.
	frameWriteTimeout = 20 * time.Second

	pingInterval = 3 * time.Second // should be 15

	inboundConn  = 1
	outboundConn = 2
)

// Config holds Server options.
type Config struct {
	// This field must be set to a valid secp256k1 private key.
	//PrivateKey *ecdsa.PrivateKey `toml:"-"`

	// MaxPeers is the maximum number of peers that can be
	// connected. It must be greater than zero.
	MaxPeers int

	// MaxPendingPeers is the maximum number of peers that can be pending in the
	// handshake phase, counted separately for inbound and outbound connections.
	// Zero defaults to preset values.
	MaxPendingPeers int `toml:",omitempty"`

	// Name sets the node name of this server.
	// Use common.MakeName to create a name that follows existing conventions.
	Name string `toml:"-"`

	// Static nodes are used as pre-configured connections which are always
	// maintained and re-connected on disconnects.
	StaticNodes []*discovery.Node

	// Protocols should contain the protocols supported
	// by the server. Matching protocols are launched for
	// each peer.
	Protocols []ProtocolInterface `toml:"-"`

	// If ListenAddr is set to a non-nil address, the server
	// will listen for incoming connections.
	ListenAddr string
}

// Server manages all p2p peer connections.
type Server struct {
	// Config fields may not be modified while the server is running.
	Config

	lock    sync.Mutex // protects running
	running bool

	ntab     discovery.Table
	listener net.Listener

	//ourHandshake *protoHandshake
	//lastLookup time.Time

	// These are for Peers, PeerCount (and nothing else).
	//peerOp     chan peerOpFunc
	//peerOpDone chan struct{}

	quit chan struct{}
	//addstatic     chan *discover.Node
	//removestatic  chan *discover.Node
	//posthandshake chan *conn
	addpeer chan *Peer
	delpeer chan *Peer
	loopWG  sync.WaitGroup // loop, listenLoop
	//	peerFeed      event.Feed

	kadPort string
	// peers map[*Peer]bool
	peers map[discovery.NodeID]*Peer
}

// Start starts running the server.
// Servers can not be re-used after stopping.
func (srv *Server) Start() (err error) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.running {
		return errors.New("server already running")
	}
	srv.running = true
	srv.peers = make(map[discovery.NodeID]*Peer)

	//log.Info("Starting P2P networking")

	// static fields
	/*if srv.PrivateKey == nil {
		return fmt.Errorf("Server.PrivateKey must be set to a non-nil key")
	}*/

	srv.quit = make(chan struct{})
	srv.addpeer = make(chan *Peer)
	srv.delpeer = make(chan *Peer)
	//srv.posthandshake = make(chan *conn)
	//srv.addstatic = make(chan *discover.Node)
	//	srv.removestatic = make(chan *discover.Node)
	//	srv.peerOp = make(chan peerOpFunc)
	//srv.peerOpDone = make(chan struct{})

	// node table

	/*ntab, err := discover.ListenUDP(srv.PrivateKey, srv.ListenAddr, srv.NAT, srv.NodeDatabase, srv.NetRestrict)
	if err != nil {
		return err
	}

	srv.ntab = ntab*/
	//	dynPeers := (srv.MaxPeers + 1) / 2
	srv.kadPort = "9001"
	discovery.StartServer(srv.kadPort)

	if err := srv.startListening(); err != nil {
		return err
	}

	for _, proto := range srv.Protocols {
		go func() {
			srv.loopWG.Add(1)
			proto.Run()
			//fmt.Println(proto.GetBaseProtocol())
			srv.loopWG.Done()
		}()
	}
	fmt.Println("Start running...")
	srv.loopWG.Add(1)
	go srv.run()
	srv.running = true

	return nil
}

func (srv *Server) run() {
	defer srv.loopWG.Done()
	peers := srv.peers

running:
	for {
		srv.scheduleTasks()
		select {
		case <-srv.quit:
			// The server was stopped. Run the cleanup logic.
			break running
		case c := <-srv.addpeer:
			fmt.Println("<-srv.addpeer", c)
			_, ok := peers[c.node.ID]
			if ok {
				// already connected
				c.Disconnect(10)
			} else {
				peers[c.node.ID] = c
			}

		case pd := <-srv.delpeer:
			// A peer disconnected.
			//d := (mclock.Now() - pd.created) / 1000
			//Debug("Removing p2p peer", "duration", d, "peers", len(peers)-1, "err", pd.err)
			curPeer, ok := peers[pd.node.ID]
			if ok && curPeer == pd {
				fmt.Println("p2p.server run. delpeer recved. peer match. remove peer", pd)
				delete(peers, pd.node.ID)
			} else {
				fmt.Println("p2p.server run. delpeer recved. peer not match")
			}
		}
	}

	//Trace("P2P networking is spinning down")

	// Terminate discovery. If there is a running lookup it will terminate soon.
	/*if srv.ntab != nil {
		srv.ntab.Close()
	}*/

	// Disconnect all peers.
	for _, p := range peers {
		p.Disconnect(10)
	}

	for len(peers) > 0 {
		p := <-srv.delpeer
		//Trace("<-delpeer (spindown)", "remainingTasks", len(runningTasks))
		delete(peers, p.node.ID)
	}
}

//scheduleTasks
func (srv *Server) scheduleTasks() {
	// TODO select nodes from ntab to connect
	fmt.Println("scheduleTasks called...")
	for _, node := range srv.StaticNodes {
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
	}
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

// listenLoop runs in its own goroutine and accepts
// inbound connections.
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
				//Debug("Temporary read error", "err", err)
				continue
			} else if err != nil {
				//Debug("Read error", "err", err)
				return
			}
			break
		}
		fmt.Println("srv.listener.Accept ok")
		go func() {
			srv.setupConn(fd, inboundConn, nil)
			slots <- struct{}{}
		}()
	}
}

// setupConn todo handshake.
func (srv *Server) setupConn(fd net.Conn, flags int, dialDest *discovery.Node) error {
	peer := &Peer{
		fd:       fd,
		created:  monotime.Now(),
		disc:     make(chan uint),
		closed:   make(chan struct{}),
		protoMap: make(map[uint16]*Protocol),
		capMap:   make(map[string]uint16),
		node:     dialDest,
	}

	var caps []Cap
	for _, proto := range srv.Protocols {
		caps = append(caps, proto.GetBaseProtocol().cap())
	}
	hsMsg := &msg{
		protoCode: ctlProtoCode,
		Message: Message{
			msgCode: ctlMsgProtoHandshake,
		},
	}
	//buffer := new(bytes.Buffer)
	buffer, err := common.Encoding(&caps)
	if err != nil {
		fd.Close()
		return err
	}
	hsMsg.payload = make([]byte, len(buffer))
	copy(hsMsg.payload, buffer)
	hsMsg.size = uint32(len(hsMsg.payload))
	fmt.Println("setupConn before sendRawMsg", hsMsg)
	//hsMsg.payload, hsMsg.size = buffer.Bytes(), uint32(buffer.Len())
	peer.sendRawMsg(hsMsg)

	msgRecv, err := peer.recvRawMsg()
	fmt.Println("setupConn after recvRawMsg", msgRecv)
	if err != nil {
		fd.Close()
		return err
	}

	var remoteCaps []Cap
	if err := common.Decoding(msgRecv.payload, &remoteCaps); err != nil {
		fd.Close()
		return err
	}

	//TODO need merge caps
	protoCode := uint16(baseProtoCode)
	for _, proto := range srv.Protocols {
		peer.protoMap[protoCode] = proto.GetBaseProtocol()
		baseProtocol := proto.GetBaseProtocol()
		myCap := baseProtocol.cap()
		str1 := myCap.String()
		fmt.Println(str1)
		peer.capMap[proto.GetBaseProtocol().cap().String()] = protoCode
		protoCode++
	}

	// TODO get Node from ntab, according nodeID
	if flags == inboundConn {
		nodeID1, _ := discovery.BytesTOID([]byte("1234567890123456789012345678901234567890123456789012345678901235"))
		addr1, _ := net.ResolveUDPAddr("udp4", "182.87.223.29:39009")
		peer.node = discovery.NewNode(nodeID1, addr1)
	}
	fmt.Println("srv.addpeer <- peer", peer)
	go func() {
		srv.loopWG.Add(1)
		srv.addpeer <- peer
		peer.run()
		srv.delpeer <- peer
		srv.loopWG.Done()
	}()
	return nil
}
