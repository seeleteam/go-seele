/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

const (
	// Maximun number of peers that can be connected
	defaultMaxPeers = 500

	// Maximum number of concurrently handshaking inbound connections.
	maxAcceptConns = 50

	defaultDialTimeout = 15 * time.Second

	// Maximum amount of time allowed for writing some bytes, not a complete message, because the message length is very highly variable.
	connWriteTimeout = 10 * time.Second

	// Maximum time allowed for reading a complete message.
	frameReadTimeout = 30 * time.Second

	inboundConn  = 1
	outboundConn = 2

	// In transfering handshake msg, length of extra data
	extraDataLen = 24
)

//P2PConfig is the Configuration of p2p
type Config struct {
	// p2p.server will listen for incoming tcp connections. And it is for udp address used for Kad protocol
	ListenAddr string `json:"address"`

	// NetworkID used to define net type, for example main net and test net.
	NetworkID uint64 `json:"networkID"`

	// static nodes which will be connected to find more nodes when the node starts
	StaticNodes []*discovery.Node `json:"staticNodes"`

	// SubPrivateKey which will be make PrivateKey
	SubPrivateKey string `json:"privateKey"`

	// PrivateKey private key for p2p module, do not use it as any accounts
	PrivateKey *ecdsa.PrivateKey
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

	loopWG sync.WaitGroup // loop, listenLoop

	peerSet  *peerSet
	peerLock sync.Mutex // lock for peer set
	log      *log.SeeleLog

	// MaxPeers max number of peers that can be connected
	MaxPeers int

	// MaxPendingPeers is the maximum number of peers that can be pending in the
	// handshake phase, counted separately for inbound and outbound connections.
	// Zero defaults to preset values.
	MaxPendingPeers int

	// Protocols should contain the protocols supported by the server.
	Protocols []Protocol

	SelfNode *discovery.Node

	genesis core.GenesisInfo
}

// NewServer initialize a server
func NewServer(genesis core.GenesisInfo, config Config, protocols []Protocol) *Server {
	return &Server{
		Config:          config,
		running:         false,
		log:             log.GetLogger("p2p"),
		MaxPeers:        defaultMaxPeers,
		quit:            make(chan struct{}),
		peerSet:         NewPeerSet(),
		MaxPendingPeers: 0,
		Protocols:       protocols,
		genesis:         genesis,
	}
}

// PeerCount return the count of peers
func (srv *Server) PeerCount() int {
	return srv.peerSet.count()
}

// Start starts running the server.
func (srv *Server) Start(nodeDir string, shard uint) (err error) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.running {
		return errors.New("server already running")
	}

	srv.running = true
	srv.log.Debug("Starting P2P networking...")
	// self node
	address := crypto.GetAddress(&srv.PrivateKey.PublicKey)
	addr, err := net.ResolveUDPAddr("udp", srv.ListenAddr)

	srv.SelfNode = discovery.NewNodeWithAddr(*address, addr, shard)
	if err != nil {
		return err
	}

	srv.log.Info("p2p.Server.Start: MyNodeID [%s]", srv.SelfNode)
	srv.kadDB = discovery.StartService(nodeDir, *address, addr, srv.StaticNodes, shard)
	srv.kadDB.SetHookForNewNode(srv.addNode)
	srv.kadDB.SetHookForDeleteNode(srv.deleteNode)

	if err := srv.startListening(); err != nil {
		return err
	}

	srv.loopWG.Add(1)
	go srv.run()
	srv.running = true
	return nil
}

func (srv *Server) addNode(node *discovery.Node) {
	if node.Shard == discovery.UndefinedShardNumber {
		return
	}

	srv.log.Debug("got discovery a new node event, node info:%s", node)
	if srv.checkPeerExist(node.ID) {
		return
	}

	//TODO UDPPort==> TCPPort
	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", node.IP.String(), node.UDPPort))
	if err != nil {
		srv.log.Error("failed to resolve tpc address %s", err)
		return
	}

	conn, err := net.DialTimeout("tcp", addr.String(), defaultDialTimeout)
	if err != nil {
		srv.log.Error("connect to a new node err: %s, node: %s", err, node)
		if conn != nil {
			conn.Close()
		}

		return
	}

	srv.log.Info("connect to a node with %s -> %s", conn.LocalAddr(), conn.RemoteAddr())
	if err := srv.setupConn(conn, outboundConn, node); err != nil {
		srv.log.Info("failed to add new node. err=%s", err)
	}
}

func (srv *Server) deleteNode(node *discovery.Node) {
	srv.deletePeer(node.ID)
}

func (srv *Server) checkPeerExist(id common.Address) bool {
	srv.peerLock.Lock()
	srv.peerLock.Unlock()

	peer := srv.peerSet.find(id)
	return peer != nil
}

func (srv *Server) addPeer(p *Peer) bool {
	srv.peerLock.Lock()
	defer srv.peerLock.Unlock()

	if p.getShardNumber() == discovery.UndefinedShardNumber {
		srv.log.Warn("got invalid peer with shard 0, peer info %s", p.Node)
		return false
	}

	peer := srv.peerSet.find(p.Node.ID)
	if peer != nil {
		srv.log.Debug("peer is already exist %s -> %s, skip %s -> %s", peer.LocalAddr(), peer.RemoteAddr(),
			p.LocalAddr(), p.RemoteAddr())
		return false
	}

	srv.peerSet.add(p)
	srv.log.Info("add peer to server, len(peers)=%d. peer %s", srv.PeerCount(), p.Node)
	p.notifyProtocolsAddPeer()

	metricsAddPeerMeter.Mark(1)
	metricsPeerCountGauge.Update(int64(srv.PeerCount()))
	return true
}

func (srv *Server) deletePeer(id common.Address) {
	srv.peerLock.Lock()
	defer srv.peerLock.Unlock()

	p := srv.peerSet.find(id)
	if p != nil {
		srv.peerSet.delete(p)
		p.notifyProtocolsDeletePeer()
		srv.log.Info("server.run delPeerChan received. peer match. remove peer. peers num=%d", srv.PeerCount())

		metricsDeletePeerMeter.Mark(1)
		metricsPeerCountGauge.Update(int64(srv.PeerCount()))
	} else {
		srv.log.Info("server.run delPeerChan received. peer not match")
	}
}

func (srv *Server) run() {
	defer srv.loopWG.Done()
	srv.log.Info("p2p start running...")

running:
	for {
		select {
		case <-srv.quit:
			srv.log.Warn("server got quit signal, run cleanup logic")
			break running
		}
	}

	// Disconnect all peers.
	srv.peerSet.foreach(func(p *Peer) {
		p.Disconnect(discServerQuit)
	})
}

func (srv *Server) startListening() error {
	// Launch the TCP listener.
	listener, err := net.Listen("tcp", srv.Config.ListenAddr)
	if err != nil {
		return err
	}

	srv.listener = listener
	srv.loopWG.Add(1)
	go srv.listenLoop()
	return nil
}

type tempError interface {
	Temporary() bool
}

// Wait wait for server until it exit
func (srv *Server) Wait() {
	srv.loopWG.Wait()
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
			srv.log.Info("Accept new connection from, %s -> %s", fd.RemoteAddr(), fd.LocalAddr())
			err := srv.setupConn(fd, inboundConn, nil)
			if err != nil {
				srv.log.Info("setupConn err, %s", err)
			}

			slots <- struct{}{}
		}()
	}
}

// setupConn Confirm both side are valid peers, have sub-protocols supported by each other
// Assume the inbound side is server side; outbound side is client side.
func (srv *Server) setupConn(fd net.Conn, flags int, dialDest *discovery.Node) error {
	srv.log.Info("setup connection with peer %s", dialDest)
	peer := NewPeer(&connection{fd: fd}, srv.Protocols, srv.log, dialDest)
	var caps []Cap
	for _, proto := range srv.Protocols {
		caps = append(caps, proto.cap())
	}

	recvMsg, _, err := srv.doHandShake(caps, peer, flags, dialDest)
	if err != nil {
		srv.log.Info("failed to do handshake with peer %s, err info %s", dialDest, err)
		peer.close()
		return err
	}

	srv.log.Debug("handshake succeed. %s -> %s", fd.LocalAddr(), fd.RemoteAddr())
	peerNodeID := recvMsg.NodeID
	if flags == inboundConn {
		peerNode, ok := srv.kadDB.FindByNodeID(peerNodeID)
		if !ok {
			srv.log.Warn("p2p.setupConn conn handshaked, not found nodeID")
			peer.close()
			return errors.New("not found nodeID in discovery database")
		}

		srv.log.Info("p2p.setupConn peerNodeID found in nodeMap. %s", peerNode.ID.ToHex())
		peer.Node = peerNode
	}

	go func() {
		srv.loopWG.Add(1)
		if srv.addPeer(peer) {
			peer.run()
			srv.deletePeer(peer.Node.ID)
		} else {
			peer.close()
		}

		srv.loopWG.Done()
	}()

	return nil
}

func (srv *Server) peerIsValidate(recvMsg *ProtoHandShake) bool {
	var genesis core.GenesisInfo
	err := json.Unmarshal(recvMsg.Params, &genesis)
	if err != nil {
		return false
	}

	for key, val := range genesis.Accounts {
		v, ok := srv.genesis.Accounts[key]
		if !ok {
			return false
		}
		if val.Cmp(v) != 0 {
			return false
		}
	}

	if srv.Config.NetworkID != recvMsg.NetworkID {
		return false
	}

	var caps []Cap
	for _, proto := range srv.Protocols {
		caps = append(caps, proto.cap())
	}
	if len(caps) != len(recvMsg.Caps) {
		return false
	}

	var str string
	var tag = true
	len := len(caps)
	for i := 0; i < len; i++ {
		str = caps[i].String()
		for j := 0; j < len; j++ {
			if recvMsg.Caps[j].String() != str {
				tag = false
				continue
			}
			tag = true
			break
		}
		if !tag {
			break
		}
	}
	return tag
}

// doHandShake Communicate each other
func (srv *Server) doHandShake(caps []Cap, peer *Peer, flags int, dialDest *discovery.Node) (recvMsg *ProtoHandShake, nounceCnt uint64, err error) {
	var renounceCnt uint64
	handshakeMsg := &ProtoHandShake{Caps: caps}
	handshakeMsg.NetworkID = srv.Config.NetworkID
	params, err := json.Marshal(srv.genesis)
	if err != nil {
		return nil, 0, err
	}
	handshakeMsg.Params = params
	nodeID := srv.SelfNode.ID
	copy(handshakeMsg.NodeID[0:], nodeID[0:])
	if flags == outboundConn {
		// client side. Send msg first
		binary.Read(rand.Reader, binary.BigEndian, &nounceCnt)
		wrapMsg, err := srv.packWrapHSMsg(handshakeMsg, dialDest.ID[0:], nounceCnt)
		if err != nil {
			return nil, 0, err
		}
		if err = peer.rw.WriteMsg(wrapMsg); err != nil {
			return nil, 0, err
		}
		recvWrapMsg, err := peer.rw.ReadMsg()
		if err != nil {
			return nil, 0, err
		}
		recvMsg, renounceCnt, err = srv.unPackWrapHSMsg(recvWrapMsg)
		if err != nil {
			return nil, 0, err
		}
		if renounceCnt != nounceCnt {
			return nil, 0, errors.New("client nounceCnt is changed")
		}
		if !srv.peerIsValidate(recvMsg) {
			return nil, 0, errors.New("node is not consitent with groups")
		}
	} else {
		// server side. Receive handshake msg first
		recvWrapMsg, err := peer.rw.ReadMsg()
		if err != nil {
			return nil, 0, err
		}
		recvMsg, nounceCnt, err = srv.unPackWrapHSMsg(recvWrapMsg)
		if err != nil {
			return nil, 0, err
		}
		if !srv.peerIsValidate(recvMsg) {
			return nil, 0, errors.New("node is not consitent with groups")
		}
		wrapMsg, err := srv.packWrapHSMsg(handshakeMsg, recvMsg.NodeID[0:], nounceCnt)
		if err != nil {
			return nil, 0, err
		}
		if err = peer.rw.WriteMsg(wrapMsg); err != nil {
			return nil, 0, err
		}
	}
	return
}

// packWrapHSMsg compose the wrapped send msg.
// A 32 byte ExtraData is used for verification process.
func (srv *Server) packWrapHSMsg(handshakeMsg *ProtoHandShake, peerNodeID []byte, nounceCnt uint64) (Message, error) {
	// Serialize should handle big-endian
	hdmsgRLP, err := common.Serialize(handshakeMsg)

	if err != nil {
		return Message{}, err
	}
	wrapMsg := Message{
		Code: ctlMsgProtoHandshake,
	}
	md5Inst := md5.New()
	if _, err := md5Inst.Write(hdmsgRLP); err != nil {
		return Message{}, err
	}
	extBuf := make([]byte, extraDataLen)

	// first 16 bytes, contains md5sum of hdmsgRLP;
	// then 8 bytes for client side nounce;
	copy(extBuf, md5Inst.Sum(nil))
	binary.BigEndian.PutUint64(extBuf[16:], nounceCnt)

	// Sign with local privateKey first
	signature := crypto.MustSign(srv.PrivateKey, crypto.MustHash(extBuf).Bytes())
	enc := make([]byte, extraDataLen+len(signature.Sig))
	copy(enc, extBuf)
	copy(enc[extraDataLen:], signature.Sig)

	// Format of wrapMsg payload, [handshake's rlp body, encoded extra data, length of encoded extra data]
	size := uint32(len(hdmsgRLP) + len(enc) + 4)
	wrapMsg.Payload = make([]byte, size)
	copy(wrapMsg.Payload, hdmsgRLP)
	copy(wrapMsg.Payload[len(hdmsgRLP):], enc)
	binary.BigEndian.PutUint32(wrapMsg.Payload[len(hdmsgRLP)+len(enc):], uint32(len(enc)))
	return wrapMsg, nil
}

// unPackWrapHSMsg verify received msg, and recover the handshake msg
func (srv *Server) unPackWrapHSMsg(recvWrapMsg Message) (recvMsg *ProtoHandShake, nounceCnt uint64, err error) {
	size := uint32(len(recvWrapMsg.Payload))
	if size < extraDataLen+4 {
		err = errors.New("received msg with invalid length")
		return
	}

	extraEncLen := binary.BigEndian.Uint32(recvWrapMsg.Payload[size-4:])
	recvHSMsgLen := size - extraEncLen - 4
	nounceCnt = binary.BigEndian.Uint64(recvWrapMsg.Payload[recvHSMsgLen+16:])
	recvEnc := recvWrapMsg.Payload[recvHSMsgLen : size-4]
	recvMsg = &ProtoHandShake{}
	if err = common.Deserialize(recvWrapMsg.Payload[:recvHSMsgLen], recvMsg); err != nil {
		return
	}
	// verify signature
	sig := crypto.Signature{
		Sig: recvEnc[extraDataLen:],
	}

	if !sig.Verify(recvMsg.NodeID, crypto.MustHash(recvEnc[0:extraDataLen]).Bytes()) {
		err = errors.New("unPackWrapHSMsg: received public key not match")
		return
	}

	// verify recvMsg's payload md5sum to prevent modification
	md5Inst := md5.New()
	if _, err = md5Inst.Write(recvWrapMsg.Payload[:recvHSMsgLen]); err != nil {
		return
	}

	if !bytes.Equal(md5Inst.Sum(nil), recvEnc[:16]) {
		err = errors.New("unPackWrapHSMsg: received md5sum not match")
		return
	}

	srv.log.Debug("unPackWrapHSMsg: verify OK!")
	return
}

// Stop terminates the execution of the p2p server
func (srv *Server) Stop() {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	if !srv.running {
		return
	}
	srv.running = false

	if srv.listener != nil {
		srv.listener.Close()
	}

	close(srv.quit)
	srv.Wait()
}

// PeerInfos array of PeerInfo for sort alphabetically by node identifier
type PeerInfos []PeerInfo

func (p PeerInfos) Len() int           { return len(p) }
func (p PeerInfos) Less(i, j int) bool { return p[i].ID < p[j].ID }
func (p PeerInfos) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// PeersInfo returns an array of metadata objects describing connected peers.
func (srv *Server) PeersInfo() []PeerInfo {
	infos := make([]PeerInfo, 0, srv.PeerCount())
	srv.peerSet.foreach(func(peer *Peer) {
		if peer != nil {
			peerInfo := peer.Info()
			infos = append(infos, *peerInfo)
		}
	})

	sort.Sort(PeerInfos(infos))
	return infos
}
