/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"bytes"
	"crypto/md5"

	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/aristanetworks/goarista/monotime"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/crypto/ecies"
	"github.com/seeleteam/go-seele/crypto/secp256k1"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p/discovery"
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

	// In transfering handshake msg, length of extra data
	hsExtraDataLen = 32
)

// Config holds Server options.
type Config struct {
	// Name node's name
	Name string //`toml:"-"`

	// ECDSAKey dumped string of Node's ecdsa.PrivateKey
	ECDSAKey string

	// PrivateKey Node's ecdsa.PrivateKey
	PrivateKey *ecdsa.PrivateKey

	// MyNodeID public key extracted from PrivateKey, so need not load from config
	MyNodeID string `toml:"-"`

	// MaxPendingPeers is the maximum number of peers that can be pending in the
	// handshake phase, counted separately for inbound and outbound connections.
	// Zero defaults to preset values.
	MaxPendingPeers int `toml:",omitempty"`

	// pre-configured nodes.
	StaticNodes []*discovery.Node

	// KadPort udp port for Kad network
	KadPort string

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

	srv.PrivateKey, err = crypto.LoadECDSAFromString(srv.ECDSAKey)
	if err != nil {
		return err
	}
	srv.MyNodeID = crypto.PubkeyToString(&srv.PrivateKey.PublicKey)
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%s", srv.KadPort))
	if err != nil {
		return err
	}
	srv.log.Info("p2p.Server.Start: MyNodeID [%s][%s]", srv.MyNodeID, addr)
	srv.kadDB = discovery.StartService(common.HexToAddress(srv.MyNodeID), addr, srv.StaticNodes)
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
				srv.log.Info("server.run  <-srv.addpeer ", c.node.ID)
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
		/*fmt.Println("\tscheduleTasks, curNode=", node.ID)
		for _, p := range srv.peers {
			fmt.Println("\tpeer's node = ", p.node.ID)
		}*/
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
		go func() {
			if err := srv.setupConn(conn, outboundConn, node); err != nil {
				srv.log.Info("scheduleTasks. setupConn called err returns. err=%s", err)
			}
		}()
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
			srv.log.Info("Accept new connection from, %s", fd.RemoteAddr())
			err := srv.setupConn(fd, inboundConn, nil)
			srv.log.Info("setupConn err, %s", err)
			slots <- struct{}{}
		}()
	}
}

// setupConn Confirm both side are valid peers, have sub-protocols supported by each other
// Assume the inbound side is server side; outbound side is client side.
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

	recvMsg, nounceCnt, nounceSvr, err := srv.doHandShake(caps, peer, flags, dialDest)
	if err != nil {
		peer.close()
		return err
	}

	peerCaps, peerNodeID := recvMsg.Caps, recvMsg.NodeID
	// TODO need merge caps and order by cap name, make sure having the same order at each end
	// TODO compute a secret key by nounceCnt, nounceSvr
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
		srv.log.Info("p2p.setupConn peerNodeID found in nodeMap. %s", peerNode.ID)
		peer.node = peerNode
	}

	srv.log.Info("p2p.setupConn conn handshaked. nounceCnt=%d nounceSvr=%d peerCaps=%s", nounceCnt, nounceSvr, peerCaps)
	go func() {
		srv.loopWG.Add(1)
		srv.addpeer <- peer
		peer.run()
		srv.delpeer <- peer
		srv.loopWG.Done()
	}()

	return nil
}

// doHandShake Communicate each other
func (srv *Server) doHandShake(caps []Cap, peer *Peer, flags int, dialDest *discovery.Node) (recvMsg *ProtoHandShake, nounceCnt uint64, nounceSvr uint64, err error) {
	handshakeMsg := &ProtoHandShake{Caps: caps}
	nodeID := common.HexToAddress(srv.MyNodeID)
	copy(handshakeMsg.NodeID[0:], nodeID[0:])

	if flags == outboundConn {
		// client side. Send msg first
		binary.Read(rand.Reader, binary.BigEndian, &nounceCnt)
		wrapMsg, err := srv.packWrapHSMsg(handshakeMsg, dialDest.ID[0:], nounceCnt, nounceSvr)
		if err != nil {
			return nil, 0, 0, err
		}

		if err = peer.sendRawMsg(wrapMsg); err != nil {
			return nil, 0, 0, err
		}

		recvWrapMsg, err := peer.recvRawMsg()
		if err != nil {
			return nil, 0, 0, err
		}

		recvMsg, _, nounceSvr, err = srv.unPackWrapHSMsg(recvWrapMsg)
		if err != nil {
			return nil, 0, 0, err
		}
	} else {
		// server side. Recv handshake msg first
		binary.Read(rand.Reader, binary.BigEndian, &nounceSvr)
		recvWrapMsg, err := peer.recvRawMsg()
		if err != nil {
			return nil, 0, 0, err
		}

		recvMsg, nounceCnt, _, err = srv.unPackWrapHSMsg(recvWrapMsg)
		if err != nil {
			return nil, 0, 0, err
		}

		wrapMsg, err := srv.packWrapHSMsg(handshakeMsg, recvMsg.NodeID[0:], nounceCnt, nounceSvr)
		if err != nil {
			return nil, 0, 0, err
		}

		if err = peer.sendRawMsg(wrapMsg); err != nil {
			return nil, 0, 0, err
		}
	}
	return
}

// packWrapHSMsg compose the wrapped send msg.
// A 32 byte ExtraData is used for verification process.
func (srv *Server) packWrapHSMsg(handshakeMsg *ProtoHandShake, peerNodeID []byte, nounceCnt uint64, nounceSvr uint64) (*msg, error) {
	// Serialize should handle big-endian
	hdmsgRLP, err := common.Serialize(handshakeMsg)
	if err != nil {
		return nil, err
	}
	wrapMsg := &msg{
		protoCode: ctlProtoCode,
		msgCode:   ctlMsgProtoHandshake,
	}
	md5Inst := md5.New()
	if _, err := md5Inst.Write(hdmsgRLP); err != nil {
		return nil, err
	}
	extBuf := make([]byte, hsExtraDataLen)
	// first 16 bytes, contains md5sum of hdmsgRLP;
	// then 8 bytes for client side nounce; 8 bytes for server side nounce
	copy(extBuf, md5Inst.Sum(nil))
	binary.BigEndian.PutUint64(extBuf[16:], nounceCnt)
	binary.BigEndian.PutUint64(extBuf[24:], nounceSvr)

	// 1. Sign with local privateKey first
	priKeyLocal := math.PaddedBigBytes(srv.PrivateKey.D, 32)
	sig, err := secp256k1.Sign(extBuf, priKeyLocal)
	if err != nil {
		return nil, err
	}
	// 2. Encrypt with peer publicKey
	pubObj := crypto.ToECDSAPub(peerNodeID[0:])
	remotePub := ecies.ImportECDSAPublic(pubObj)

	encOrg := make([]byte, hsExtraDataLen+len(sig))
	copy(encOrg, extBuf)
	copy(encOrg[hsExtraDataLen:], sig)
	enc, err := ecies.Encrypt(rand.Reader, remotePub, encOrg, nil, nil)
	if err != nil {
		return nil, err
	}
	// Format of wrapMsg payload, [handshake's rlp body, encoded extra data, length of encoded extra data]
	wrapMsg.size = uint32(len(hdmsgRLP) + len(enc) + 4)
	wrapMsg.payload = make([]byte, wrapMsg.size)
	copy(wrapMsg.payload, hdmsgRLP)
	copy(wrapMsg.payload[len(hdmsgRLP):], enc)
	binary.BigEndian.PutUint32(wrapMsg.payload[len(hdmsgRLP)+len(enc):], uint32(len(enc)))
	return wrapMsg, nil
}

// unPackWrapHSMsg verify recved msg, and recover the handshake msg
func (srv *Server) unPackWrapHSMsg(recvWrapMsg *msg) (recvMsg *ProtoHandShake, nounceCnt uint64, nounceSvr uint64, err error) {
	if recvWrapMsg.size < hsExtraDataLen+4 {
		err = errors.New("recved err msg")
		return
	}
	extraEncLen := binary.BigEndian.Uint32(recvWrapMsg.payload[recvWrapMsg.size-4:])
	recvHSMsgLen := recvWrapMsg.size - extraEncLen - 4
	nounceCnt = binary.BigEndian.Uint64(recvWrapMsg.payload[recvHSMsgLen+16:])
	nounceSvr = binary.BigEndian.Uint64(recvWrapMsg.payload[recvHSMsgLen+24:])
	recvEnc := recvWrapMsg.payload[recvHSMsgLen : recvWrapMsg.size-4]

	recvMsg = &ProtoHandShake{}
	if err = common.Deserialize(recvWrapMsg.payload[:recvHSMsgLen], recvMsg); err != nil {
		return
	}

	// Decrypt with local private key, make sure it is sended to local
	eciesPriKey := ecies.ImportECDSA(srv.PrivateKey)
	encOrg, err := eciesPriKey.Decrypt(rand.Reader, recvEnc, nil, nil)
	if err != nil {
		return
	}

	// Verify peer public key, make sure it is sended from correct peer
	recvPubkey, err := secp256k1.RecoverPubkey(encOrg[0:hsExtraDataLen], encOrg[hsExtraDataLen:])
	if err != nil {
		return
	}

	if !bytes.Equal(recvMsg.NodeID[0:], recvPubkey[1:]) {
		err = errors.New("unPackWrapHSMsg: recvPubkey not match")
		return
	}

	// Verify recvMsg's payload md5sum to prevent modification
	md5Inst := md5.New()
	if _, err = md5Inst.Write(recvWrapMsg.payload[:recvHSMsgLen]); err != nil {
		return
	}

	if !bytes.Equal(md5Inst.Sum(nil), encOrg[:16]) {
		err = errors.New("unPackWrapHSMsg: recved md5sum not match!")
		return
	}
	srv.log.Info("unPackWrapHSMsg: verify OK!")
	return
}
