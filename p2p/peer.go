/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

const (
	pingInterval   = 15 * time.Second                 // ping interval for peer tcp connection. Should be 15
	discServerQuit = "disconnect because server quit" // p2p.server need quit, all peers should quit as it can
)

// Peer represents a connected remote node.
type Peer struct {
	protocolErr   chan error
	closed        chan struct{}
	Node          *discovery.Node // remote peer that this peer connects
	disconnection chan string
	protocolMap   map[string]protocolRW // protocol cap => protocol read write wrapper
	rw            *connection

	wg   sync.WaitGroup
	log  *log.SeeleLog
	lock sync.Mutex
}

// NewPeer creates and returns a new peer.
func NewPeer(conn *connection, log *log.SeeleLog, node *discovery.Node) *Peer {
	closed := make(chan struct{})

	return &Peer{
		rw:            conn,
		disconnection: make(chan string),
		closed:        closed,
		log:           log,
		protocolErr:   make(chan error),
		Node:          node,
		lock:          sync.Mutex{},
	}
}

func (p *Peer) setProtocols(protocols []Protocol) {
	offset := baseProtoCode
	protoMap := make(map[string]protocolRW)
	for _, protocol := range protocols {
		protoRW := protocolRW{
			bQuited:  false,
			rw:       p.rw,
			offset:   offset,
			Protocol: protocol,
			in:       make(chan Message, 1),
			close:    p.closed,
		}

		protoMap[protocol.cap().String()] = protoRW
		offset += protocol.Length
		p.log.Debug("setProtocols called, add protocol: %s", protocol.cap())
	}

	p.protocolMap = protoMap
}

func (p *Peer) getShardNumber() uint {
	return p.Node.Shard
}

// run assumes that SubProtocol will never quit, otherwise proto.DelPeerCh may be closed before peer.run quits?
func (p *Peer) run() (err error) {
	var readErr = make(chan error, 1)
	p.wg.Add(2)
	go p.readLoop(readErr)
	go p.pingLoop()

	// Wait for an error or disconnect.
errLoop:
	for {
		select {
		case err = <-readErr:
			p.log.Warn("p2p.peer.run read err %s", err)
			break errLoop
		case reason := <-p.disconnection:
			p.log.Info("p2p peer got disconnection request")
			err = fmt.Errorf("disconnection error received, %s", reason)
			break errLoop
		case err = <-p.protocolErr:
			if err != nil {
				p.log.Warn("p2p peer got protocol err %s", err.Error())
			}
			break errLoop
		}
	}
	p.close()
	p.wg.Wait()
	p.log.Info("p2p.peer.run quit. err=%s", err)
	return err
}

func (p *Peer) close() {
	p.lock.Lock()
	defer p.lock.Unlock()

	close(p.protocolErr)
	close(p.closed)
	close(p.disconnection)
	p.disconnection = nil
	p.rw.fd.Close()
}

func (p *Peer) pingLoop() {
	ping := time.NewTimer(pingInterval)
	defer p.log.Debug("exit ping loop.")
	defer p.wg.Done()
	defer ping.Stop()
	for {
		select {
		case <-ping.C:
			p.sendCtlMsg(ctlMsgPingCode)
			ping.Reset(pingInterval)
		case <-p.closed:
			return
		}
	}
}

func (p *Peer) readLoop(readErr chan<- error) {
	defer p.log.Debug("exit read loop, remote: %s", p.RemoteAddr())
	defer p.wg.Done()
	for {
		msgRecv, err := p.rw.ReadMsg()
		if err != nil {
			readErr <- err
			return
		}

		if err = p.handle(msgRecv); err != nil {
			readErr <- err
			return
		}
	}
}

func (p *Peer) notifyProtocolsAddPeer() {
	p.wg.Add(len(p.protocolMap))
	p.log.Info("notifyProtocolsAddPeer called, len(protocolMap)= %d, %s -> %s", len(p.protocolMap), p.LocalAddr(), p.RemoteAddr())
	for _, proto := range p.protocolMap {
		go func(proto protocolRW) {
			defer p.wg.Done()

			if proto.AddPeer != nil {
				p.log.Debug("protocol.AddPeer called. protocol: %s", proto.cap())
				if !proto.AddPeer(p, &proto) {
					proto.bQuited = true

					// seele protocol is the highest weight, tcp of peer need to be closed
					if proto.Name == common.SeeleProtoName {
						p.log.Debug("notifyProtocolsAddPeer AddPeer err got. name=%s node=%s", proto.Name, fmt.Sprintf("%x", p.Node.ID))
						// close connection of peer
						p.Disconnect("Seeleproto addpeer err, close connection")
					}
				}
			}
		}(proto)
	}
}

func (p *Peer) notifyProtocolsDeletePeer() {
	p.wg.Add(len(p.protocolMap))
	p.log.Debug("notifyProtocolsDeletePeer called, len(protocolMap)=%d", len(p.protocolMap))
	for _, proto := range p.protocolMap {
		go func(proto protocolRW) {
			defer p.wg.Done()

			if proto.DeletePeer != nil {
				p.log.Debug("protocol.DeletePeer called. protocol:%s", proto.cap())
				proto.DeletePeer(p)
			}
		}(proto)
	}
}

func (p *Peer) handle(msgRecv *Message) error {
	// control msg
	if msgRecv.Code < baseProtoCode {
		switch {
		case msgRecv.Code == ctlMsgPingCode:
			go p.sendCtlMsg(ctlMsgPongCode)
		case msgRecv.Code == ctlMsgPongCode:
			//p.log.Debug("peer handle Ping msg.")
			return nil
		case msgRecv.Code == ctlMsgDiscCode:
			return fmt.Errorf("error=%d", ctlMsgDiscCode)
		}

		return nil
	}

	var protocolTarget protocolRW
	found := false
	for _, p := range p.protocolMap {
		if msgRecv.Code >= p.offset && msgRecv.Code < p.offset+p.Length {
			protocolTarget = p
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf(fmt.Sprintf("could not found mapping proto with code %d", msgRecv.Code))
	}

	if !protocolTarget.bQuited {
		protocolTarget.in <- *msgRecv
	}

	return nil
}

func (p *Peer) sendCtlMsg(msgCode uint16) error {
	hsMsg := Message{
		Code: msgCode,
	}

	p.rw.WriteMsg(&hsMsg)

	return nil
}

// Disconnect terminates the peer connection with the given reason.
// It returns immediately and does not wait until the connection is closed.
func (p *Peer) Disconnect(reason string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.disconnection != nil {
		p.disconnection <- reason
	}
}

type protocolRW struct {
	Protocol
	bQuited bool
	offset  uint16
	in      chan Message // read message channel, message will be transferred here when it is a protocol message
	rw      MsgReadWriter
	close   chan struct{}
}

func (rw *protocolRW) WriteMsg(msg *Message) (err error) {
	if msg.Code >= rw.Length {
		return errors.New("invalid msg code")
	}

	msg.Code += rw.offset

	return rw.rw.WriteMsg(msg)
}

func (rw *protocolRW) ReadMsg() (*Message, error) {
	select {
	case msg := <-rw.in:
		msg.Code -= rw.offset

		return &msg, nil
	case <-rw.close:
		return &Message{}, errors.New("peer connection closed")
	}
}

// RemoteAddr returns the remote address of the network connection.
func (p *Peer) RemoteAddr() net.Addr {
	return p.rw.fd.RemoteAddr()
}

// LocalAddr returns the local address of the network connection.
func (p *Peer) LocalAddr() net.Addr {
	return p.rw.fd.LocalAddr()
}

// PeerInfo represents a short summary of a connected peer
type PeerInfo struct {
	ID      string   `json:"id"`   // Unique of the node
	Caps    []string `json:"caps"` // Sum-protocols advertised by this particular peer
	Network struct {
		LocalAddress  string `json:"localAddress"`  // Local endpoint of the TCP data connection
		RemoteAddress string `json:"remoteAddress"` // Remote endpoint of the TCP data connection
	} `json:"network"`
	Protocols map[string]interface{} `json:"protocols"` // Sub-protocol specific metadata fields
	Shard     uint                   `json:"shard"`     // shard id of the node
}

// Info returns data of the peer but not contain id and name.
func (p *Peer) Info() *PeerInfo {
	var caps []string
	protocols := make(map[string]interface{})

	for cap, protocol := range p.protocolMap {
		caps = append(caps, cap)

		protoInfo := interface{}("unknown")
		if query := protocol.Protocol.GetPeer; query != nil {
			if metadata := query(p.Node.ID); metadata != nil {
				protoInfo = metadata
			} else {
				protoInfo = "handshake"
			}
		}
		protocols[protocol.Protocol.Name] = protoInfo
	}

	info := &PeerInfo{
		ID:        p.Node.ID.Hex(),
		Caps:      caps,
		Protocols: protocols,
		Shard:     p.getShardNumber(),
	}
	info.Network.LocalAddress = p.LocalAddr().String()
	info.Network.RemoteAddress = p.RemoteAddr().String()

	return info
}
