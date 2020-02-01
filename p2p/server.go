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
	"errors"
	"fmt"
	"math/big"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/sirupsen/logrus"
	set "gopkg.in/fatih/set.v0"
)

const (
	// Maximum number of peers that can be connected for each shard
	maxConnsPerShard = 40

	// Maximum number of peers that node actively connects to.
	maxActiveConnsPerShard = 25

	defaultDialTimeout = 15 * time.Second

	// Maximum amount of time allowed for writing some bytes, not a complete message, because the message length is very highly variable.
	connWriteTimeout = 15 * time.Second

	// Maximum time allowed for reading a complete message.
	frameReadTimeout = 25 * time.Second

	// interval to select new node to connect from the free node list.
	checkConnsNumInterval = 7 * time.Second
	inboundConn           = 1
	outboundConn          = 2

	// In transferring handshake msg, length of extra data
	extraDataLen = 24

	// Minimum recommended number of peers of one shard
	minNumOfPeerPerShard = uint(2)

	// maxConnectionsPerIp represents max connections that node from one ip can connect to.
	// Reject connections if  ipSet[ip] > maxConnectionsPerIp.
	maxConnsPerShardPerIp = uint(maxConnsPerShard / 2)
)

// Config is the Configuration of p2p
type Config struct {
	// p2p.server will listen for incoming tcp connections. And it is for udp address used for Kad protocol
	ListenAddr string `json:"address"`

	// NetworkID used to define net type, for example main net and test net.
	NetworkID string `json:"networkID"`

	// static nodes which will be connected to find more nodes when the node started
	StaticNodes []*discovery.Node `json:"staticNodes"`

	// SubPrivateKey which will be make PrivateKey
	SubPrivateKey string `json:"privateKey"`

	// PrivateKey private key for p2p module, do not use it as any accounts
	PrivateKey *ecdsa.PrivateKey `json:"-"`
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

	nodeSet  *nodeSet
	peerSet  *peerSet
	peerLock sync.Mutex // lock for peer set
	log      *log.SeeleLog

	// MaxPendingPeers is the maximum number of peers that can be pending in the
	// handshake phase, counted separately for inbound and outbound connections.
	// Zero defaults to preset values.
	MaxPendingPeers int

	// Protocols should contain the protocols supported by the server.
	Protocols []Protocol

	SelfNode *discovery.Node

	genesis core.GenesisInfo

	// genesisHash is used for handshake
	genesisHash common.Hash

	// maxConnections represents max connections that node can connect to.
	// Reject connections if srv.PeerCount > maxConnections.
	maxConnections int

	// maxActiveConnections represents max connections that node can actively connect to.
	// Need not connect to a new node if srv.PeerCount > maxActiveConnections.
	maxActiveConnections int

	peerNumLock sync.Mutex // lock for num of peers per shard
}

// NewServer initialize a server
func NewServer(genesis core.GenesisInfo, config Config, protocols []Protocol) *Server {
	// add genesisHash with shard set to 0 to calculate hash
	shard := genesis.ShardNumber
	genesis.ShardNumber = 0

	// set the master account and balance to empty to calculate hash
	masteraccount := genesis.Masteraccount
	balance := genesis.Balance
	genesis.Masteraccount, _ = common.HexToAddress("0x0000000000000000000000000000000000000000")
	genesis.Balance = big.NewInt(0)

	hash := genesis.Hash()
	genesis.ShardNumber = shard
	genesis.Masteraccount = masteraccount
	genesis.Balance = balance

	return &Server{
		Config:               config,
		running:              false,
		log:                  log.GetLogger("p2p"),
		quit:                 make(chan struct{}),
		peerSet:              NewPeerSet(),
		nodeSet:              NewNodeSet(),
		MaxPendingPeers:      0,
		Protocols:            protocols,
		genesis:              genesis,
		genesisHash:          hash,
		maxConnections:       maxConnsPerShard * common.ShardCount,
		maxActiveConnections: maxActiveConnsPerShard * common.ShardCount,
	}
}

// PeerCount return the count of peers
func (srv *Server) PeerCount() int {
	return srv.peerSet.count()
}

// Start starts the server.
func (srv *Server) Start(nodeDir string, shard uint) (err error) {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	if srv.running {
		return errors.New("server already running")
	}

	address := crypto.GetAddress(&srv.PrivateKey.PublicKey)
	addr, err := net.ResolveUDPAddr("udp", srv.ListenAddr)
	if err != nil {
		return err
	}

	srv.log.Debug("Starting P2P networking...")
	srv.SelfNode = discovery.NewNodeWithAddr(*address, addr, shard)

	srv.log.Info("p2p.Server.Start: MyNodeID [%s]", srv.SelfNode)
	srv.kadDB = discovery.StartService(nodeDir, *address, addr, srv.Config.StaticNodes, shard)
	srv.kadDB.SetHookForNewNode(srv.addNode)
	srv.kadDB.SetHookForDeleteNode(srv.deleteNode)
	// add static nodes to srv node set;
	for _, node := range srv.Config.StaticNodes {
		if !node.ID.IsEmpty() {
			srv.nodeSet.tryAdd(node)
		}

	}
	if err := srv.startListening(); err != nil {
		return err
	}

	srv.loopWG.Add(1)
	go srv.run()
	srv.running = true

	// just in debug mode
	if srv.log.GetLevel() >= logrus.DebugLevel {
		go srv.printPeers()
	}

	return nil
}

// printPeers used print handshake peers log, not just in debug
func (srv *Server) printPeers() {
	timer := time.NewTimer(1 * time.Hour)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ticker.C:
			srv.log.Debug("handshake peers number: %d, time: %d", srv.PeerCount(), time.Now().UnixNano())
		case <-timer.C:
			break loop
		}
	}
}

func (srv *Server) addNode(node *discovery.Node) {
	if node.Shard == discovery.UndefinedShardNumber {
		return
	}
	numPeersDelete := srv.PeerCount() - srv.maxActiveConnections
	if numPeersDelete > 0 {
		for i := 0; i < numPeersDelete; i++ {
			if srv.PeerCount() > srv.maxActiveConnections {
				srv.deletePeerRand()
				//srv.log.Warn("got discovery a new node event. Reached connection limit, node:%v", node.String())
				//return
			}
		}
	}

	srv.nodeSet.tryAdd(node)
	srv.connectNode(node)
	srv.log.Debug("got discovery a new node event, node info:%s", node)

}

func (srv *Server) connectNode(node *discovery.Node) {
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
		srv.log.Debug("connect to a new node err: %s, node: %s", err, node)
		if conn != nil {
			conn.Close()
		}
		return
	}

	srv.log.Info("connect to a node with %s -> %s", conn.LocalAddr(), conn.RemoteAddr())
	if err := srv.setupConn(conn, outboundConn, node); err != nil {
		srv.log.Debug("failed to add new node. err=%s", err)
		return
	}
	return
}

func (srv *Server) deleteNode(node *discovery.Node) {
	srv.nodeSet.delete(node)
	srv.deletePeer(node.ID)
}

func (srv *Server) checkPeerExist(id common.Address) bool {
	srv.peerLock.Lock()
	defer srv.peerLock.Unlock()

	peer := srv.peerSet.find(id)
	return peer != nil
}

func (srv *Server) addPeer(p *Peer) (bool, bool) { //bool, bool: addPeer isAdd, isRun
	srv.peerLock.Lock()
	defer srv.peerLock.Unlock()

	if p.getShardNumber() == discovery.UndefinedShardNumber {
		srv.log.Warn("got invalid peer with shard 0, peer info %s", p.Node)
		return false, false
	}

	peer := srv.peerSet.find(p.Node.ID)
	if peer != nil {
		srv.log.Debug("peer is already exist %s -> %s, skip %s -> %s", peer.LocalAddr(), peer.RemoteAddr(),
			p.LocalAddr(), p.RemoteAddr())
		return false, true // find the peer, should not return false, otherwise the up layer will close this peer
	}

	srv.peerSet.add(p)
	srv.nodeSet.setNodeStatus(p.Node, true)
	srv.log.Debug("add peer to server, len(peers)=%d. peer %s", srv.PeerCount(), p.Node)
	p.notifyProtocolsAddPeer()

	metricsAddPeerMeter.Mark(1)
	metricsPeerCountGauge.Update(int64(srv.PeerCount()))
	return true, false
}

func (srv *Server) deletePeer(id common.Address) {
	srv.peerLock.Lock()
	defer srv.peerLock.Unlock()

	p := srv.peerSet.find(id)
	if p != nil {
		srv.nodeSet.setNodeStatus(p.Node, false)
		srv.peerSet.delete(p)
		p.notifyProtocolsDeletePeer()
		srv.log.Debug("server.run delPeerChan received. peer match. remove peer. peers num=%d", srv.PeerCount())

		metricsDeletePeerMeter.Mark(1)
		metricsPeerCountGauge.Update(int64(srv.PeerCount()))
	} else {
		srv.log.Info("server.run delPeerChan received. peer not match")
	}
}

func (srv *Server) deletePeerRand() {
	srv.peerLock.Lock()
	defer srv.peerLock.Unlock()

	p := srv.peerSet.getRandPeer()

	if p != nil {
		srv.nodeSet.setNodeStatus(p.Node, false)
		srv.peerSet.delete(p)
		p.notifyProtocolsDeletePeer()
		srv.log.Debug("server.run delPeerChan received. peer match. remove peer. peers num=%d", srv.PeerCount())

		metricsDeletePeerMeter.Mark(1)
		metricsPeerCountGauge.Update(int64(srv.PeerCount()))
	} else {
		srv.log.Info("server.run delPeerChan received. peer not match")
	}
}
func (srv *Server) run() {
	defer srv.loopWG.Done()
	srv.log.Info("p2p start running...")

	checkTicker := time.NewTicker(checkConnsNumInterval)
	checkTicker1 := time.NewTicker(12*checkConnsNumInterval + 3)

running:
	for {
		select {
		case <-checkTicker1.C:
			go srv.doSelectNodeToConnect()
		case <-checkTicker.C:
			if srv.nodeSet.getSelfShardNodeNum() < 2 {
				srv.log.Debug("local Node numer %d", srv.nodeSet.getSelfShardNodeNum())
				go srv.doSelectLocalNodeToConnect()
			}

		case <-srv.quit:
			srv.log.Debug("server got quit signal, run cleanup logic")
			break running
		}
	}

	// Disconnect all peers.
	peers := srv.peerSet.getPeers()
	for _, peer := range peers {
		if peer != nil {
			peer.Disconnect(discServerQuit)
		}
	}
}

// doSelectNodeToConnect selects one free node from nodeMap to connect
func (srv *Server) doSelectNodeToConnect() {

	if !srv.nodeSet.ifNeedAddNodes() {
		return
	}
	for _, node := range srv.StaticNodes {
		if node.ID.IsEmpty() || srv.checkPeerExist(node.ID) {
			continue
		} else {
			srv.connectNode(node)
		}
	}
	selectNodeSet := srv.nodeSet.randSelect()

	if selectNodeSet == nil {
		return
	}
	for i := 0; i < len(selectNodeSet); i++ {

		if selectNodeSet[i] != nil {
			srv.log.Info("p2p.server doSelectNodeToConnect. Node=%s ,%d", selectNodeSet[i].IP.String(), selectNodeSet[i].UDPPort)
			srv.connectNode(selectNodeSet[i])
		}
	}

}

func (srv *Server) doSelectLocalNodeToConnect() {
	//for _, node := range srv.StaticNodes {
	//	if node.ID.IsEmpty() || srv.checkPeerExist(node.ID) {
	//		continue
	//	} else {
	//		srv.connectNode(node)
	//	}
	//}

	selectNodeSet := srv.nodeSet.randSelect()

	if selectNodeSet == nil {
		return
	}
	for i := 0; i < len(selectNodeSet); i++ {
		node := selectNodeSet[i]
		if node != nil {
			if node.Shard == common.LocalShardNumber {
				srv.log.Info("p2p.server doSelectLocalNodeToConnect. Node=%s ,%d", selectNodeSet[i].IP.String(), selectNodeSet[i].UDPPort)
				srv.connectNode(selectNodeSet[i])
			}
		}
	}
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
	tokens := srv.maxConnections
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

			if err != nil && strings.Contains(err.Error(), "too many incomming") {
				return
			}
			slots <- struct{}{}
		}()
	}
}

// setupConn Confirm both side are valid peers, have sub-protocols supported by each other
// Assume the inbound side is server side; outbound side is client side.
func (srv *Server) setupConn(fd net.Conn, flags int, dialDest *discovery.Node) error {
	if flags == inboundConn && srv.PeerCount() > srv.maxConnections {
		srv.log.Warn("setup connection with peer %s. reached max incoming connection limit, reject!", dialDest)
		return errors.New("Too many incoming connections")
	}

	srv.log.Debug("setup connection with peer %s", dialDest)
	peer := NewPeer(&connection{fd: fd, log: srv.log}, srv.log, dialDest)
	var caps []Cap
	for _, proto := range srv.Protocols {
		caps = append(caps, proto.cap())
	}

	sort.Sort(capsByNameAndVersion(caps))
	recvMsg, _, err := srv.doHandShake(caps, peer, flags, dialDest)
	if err != nil {
		srv.log.Debug("failed to do handshake with peer %s, err info %s", dialDest, err)
		peer.close()
		return err
	}

	srv.log.Debug("handshake succeed. %s -> %s", fd.LocalAddr(), fd.RemoteAddr())
	peerNodeID := recvMsg.NodeID
	if flags == inboundConn {
		peerNode, ok := srv.kadDB.FindByNodeID(peerNodeID)
		if !ok {
			srv.log.Warn("p2p.setupConn conn handshaked, not found nodeID:%s", peerNodeID)
			peer.close()
			return errors.New("not found nodeID in discovery database")
		}

		srv.log.Info("p2p.setupConn peerNodeID found in nodeMap. %s", peerNode.ID.Hex())
		peer.Node = peerNode
	}

	go func() {
		//srv.loopWG.Add(1)
		isAdd, isRun := srv.addPeer(peer)
		if isAdd && !isRun {
			//srv.log.Error("RUN BEGIN")
			peer.run()
			//srv.deletePeer(peer.Node.ID)
			//srv.log.Error("RUN END")
		}
		if !isAdd {
			peer.close()
		}

		//srv.loopWG.Done()
	}()

	return nil
}

func (srv *Server) SetMaxConnections(maxConns int) {
	srv.maxConnections = maxConns
}

func (srv *Server) SetMaxActiveConnections(maxActiveConns int) {
	srv.maxActiveConnections = maxActiveConns
}

func (srv *Server) peerIsValidate(recvMsg *ProtoHandShake) ([]Cap, bool) {
	// validate hash of genesisHash without shard
	if !bytes.Equal(srv.genesisHash.Bytes(), recvMsg.Params) {
		return nil, false
	}

	if srv.Config.NetworkID != recvMsg.NetworkID {
		return nil, false
	}

	localCapSet := set.New()
	for _, proto := range srv.Protocols {
		localCapSet.Add(proto.cap())
	}

	var capNameList []Cap
	for _, cap := range recvMsg.Caps {
		if localCapSet.Has(cap) {
			capNameList = append(capNameList, cap)
		}
	}

	if len(capNameList) == 0 {
		return nil, false
	}

	return capNameList, true
}

func (srv *Server) getProtocolsByCaps(capList []Cap) (proList []Protocol) {
	for _, cap := range capList {
		for _, pro := range srv.Protocols {
			if pro.cap().String() == cap.String() {
				proList = append(proList, pro)
				break
			}
		}
	}

	return
}

// doHandShake Communicate each other
func (srv *Server) doHandShake(caps []Cap, peer *Peer, flags int, dialDest *discovery.Node) (recvMsg *ProtoHandShake, nounceCnt uint64, err error) {
	var renounceCnt uint64
	handshakeMsg := &ProtoHandShake{Caps: caps}
	handshakeMsg.NetworkID = srv.Config.NetworkID
	handshakeMsg.Params = srv.genesisHash.Bytes()
	nodeID := srv.SelfNode.ID
	copy(handshakeMsg.NodeID[0:], nodeID[0:])
	if flags == outboundConn {
		// client side. Send msg first
		if err := binary.Read(rand.Reader, binary.BigEndian, &nounceCnt); err != nil {
			return nil, 0, err
		}

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

		capList, bValid := srv.peerIsValidate(recvMsg)
		if !bValid {
			return nil, 0, errors.New("node is not consistent with groups")
		}

		sort.Sort(capsByNameAndVersion(capList))
		peer.setProtocols(srv.getProtocolsByCaps(capList))

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

		capList, bValid := srv.peerIsValidate(recvMsg)
		if !bValid {
			return nil, 0, errors.New("node is not consistent with groups")
		}

		sort.Sort(capsByNameAndVersion(capList))
		peer.setProtocols(srv.getProtocolsByCaps(capList))

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
func (srv *Server) packWrapHSMsg(handshakeMsg *ProtoHandShake, peerNodeID []byte, nounceCnt uint64) (*Message, error) {
	// Serialize should handle big-endian
	hdmsgRLP, err := common.Serialize(handshakeMsg)

	if err != nil {
		return &Message{}, err
	}
	wrapMsg := Message{
		Code: ctlMsgProtoHandshake,
	}
	md5Inst := md5.New()
	if _, err := md5Inst.Write(hdmsgRLP); err != nil {
		return &Message{}, err
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
	return &wrapMsg, nil
}

// unPackWrapHSMsg verify received msg, and recover the handshake msg
func (srv *Server) unPackWrapHSMsg(recvWrapMsg *Message) (recvMsg *ProtoHandShake, nounceCnt uint64, err error) {
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
	peers := srv.peerSet.getPeers()

	for _, p := range peers {
		if p != nil {
			peerInfo := p.Info()
			infos = append(infos, *peerInfo)
		}
	}

	sort.Sort(PeerInfos(infos))

	return infos
}

// IsListening return whether the node is listen or not
func (srv *Server) IsListening() bool {
	return srv.listener != nil
}
