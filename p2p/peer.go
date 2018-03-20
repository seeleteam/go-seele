/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

const (
	pingInterval         = 15 * time.Second // ping interval for peer tcp connection. Should be 15
	discAlreadyConnected = 10               // node already has connection
	discServerQuit       = 11               // p2p.server need quit, all peers should quit as it can
)

// Peer represents a connected remote node.
type Peer struct {
	err           chan error
	closed        chan struct{}
	Node          *discovery.Node // remote peer that this peer connects
	disconnection chan uint
	protocolMap   map[string]protocolRW // protocol cap => protocol read write wrapper
	rw            MsgReadWriter

	wg  sync.WaitGroup
	log *log.SeeleLog
}

func NewPeer(conn *connection, protocols []ProtocolInterface, log *log.SeeleLog, node *discovery.Node) *Peer {
	offset := baseProtoCode
	protoMap := make(map[string]protocolRW)
	for _, p := range protocols {
		proto := *p.GetBaseProtocol()

		protoRW := protocolRW{
			rw:       conn,
			offset:   offset,
			Protocol: proto,
			in:       make(chan Message),
		}

		protoMap[proto.cap().String()] = protoRW
		offset += proto.Length
	}

	return &Peer{
		rw:            conn,
		protocolMap:   protoMap,
		disconnection: make(chan uint),
		closed:        make(chan struct{}),
		log:           log,
		err:           make(chan error),
		Node:          node,
	}
}

// run assumes that SubProtocol will never quit, otherwise proto.DelPeerCh may be closed before peer.run quits?
func (p *Peer) run() {
	// add peer to protocols
	var (
		readErr = make(chan error, 1)
		err     error
	)
	for _, proto := range p.protocolMap {
		proto.AddPeerCh <- p
	}

	p.wg.Add(2)
	go p.readLoop(readErr)
	go p.pingLoop()

	// Wait for an error or disconnect.
loop:
	for {
		select {
		case err = <-readErr:
			p.log.Info("p2p.peer.run read err %s", err)
			p.err <- err
			break loop
		case <-p.disconnection:
			p.err <- errors.New("disconnection error recved")
			break loop
		}
	}

	p.close()
	p.wg.Wait()
	// send delpeer message for each protocols
	for _, proto := range p.protocolMap {
		proto.DelPeerCh <- p
	}
	p.log.Info("p2p.peer.run quit. err=%s", p.err)
}

func (p *Peer) close() {
	close(p.closed)
	close(p.disconnection)
}

func (p *Peer) pingLoop() {
	ping := time.NewTimer(pingInterval)
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

func (p *Peer) readLoop(errc chan<- error) {
	defer p.wg.Done()
	for {
		msgRecv, err := p.rw.ReadMsg()
		if err != nil {
			errc <- err
			return
		}
		if err = p.handle(msgRecv); err != nil {
			errc <- err
			return
		}
	}
}

func (p *Peer) handle(msgRecv Message) error {
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
		return errors.New(fmt.Sprintf("could not found mapping proto with code %d", msgRecv.Code))
	}

	select {
	case protocolTarget.in <- msgRecv:
		return nil
	case <-p.closed:
		return io.EOF
	}
}

func (p *Peer) sendCtlMsg(msgCode uint16) error {
	hsMsg := Message{
		Code: msgCode,
	}

	p.rw.WriteMsg(hsMsg)

	return nil
}

// Disconnect terminates the peer connection with the given reason.
// It returns immediately and does not wait until the connection is closed.
func (p *Peer) Disconnect(reason uint) {
	select {
	case p.disconnection <- reason:
	case <-p.closed:
	}
}

type protocolRW struct {
	Protocol
	offset uint16
	in     chan Message // read message channel, message will be transferred here when it is a protocol message
	rw     MsgReadWriter
}

func (rw *protocolRW) WriteMsg(msg Message) (err error) {
	if msg.Code >= rw.Length {
		return errors.New("invalid msg code")
	}

	msg.Code += rw.offset
	return rw.rw.WriteMsg(msg)
}

func (rw *protocolRW) ReadMsg() (Message, error) {
	rw.rw.ReadMsg()
	select {
	case msg := <-rw.in:
		msg.Code -= rw.offset
		return msg, nil
	}
}
