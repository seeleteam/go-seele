/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"errors"
	"fmt"
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
	protocolErr   chan error
	closed        chan struct{}
	Node          *discovery.Node // remote peer that this peer connects
	disconnection chan uint
	protocolMap   map[string]protocolRW // protocol cap => protocol read write wrapper
	rw            *connection

	wg  sync.WaitGroup
	log *log.SeeleLog
}

func NewPeer(conn *connection, protocols []Protocol, log *log.SeeleLog, node *discovery.Node) *Peer {
	offset := baseProtoCode
	protoMap := make(map[string]protocolRW)
	for _, p := range protocols {
		protoRW := protocolRW{
			rw:       conn,
			offset:   offset,
			Protocol: p,
			in:       make(chan Message, 1),
		}

		protoMap[p.cap().String()] = protoRW
		offset += p.Length
	}

	return &Peer{
		rw:            conn,
		protocolMap:   protoMap,
		disconnection: make(chan uint),
		closed:        make(chan struct{}),
		log:           log,
		protocolErr:   make(chan error),
		Node:          node,
	}
}

// run assumes that SubProtocol will never quit, otherwise proto.DelPeerCh may be closed before peer.run quits?
func (p *Peer) run() (err error) {
	var readErr = make(chan error, 1)
	p.wg.Add(2)
	go p.readLoop(readErr)
	go p.pingLoop()

	p.notifyProtocols()
	// Wait for an error or disconnect.
errLoop:
	for {
		select {
		case err = <-readErr:
			p.log.Warn("p2p.peer.run read err %s", err.Error())
			break errLoop
		case <-p.disconnection:
			p.log.Info("p2p peer got disconnection request")
			err = errors.New("disconnection error recved")
			break errLoop
		case err = <-p.protocolErr:
			p.log.Warn("p2p peer got protocol err %s", err.Error())
			break errLoop
		}
	}

	p.close()
	p.wg.Wait()
	p.log.Info("p2p.peer.run quit. err=%s", err)

	return err
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

func (p *Peer) readLoop(readErr chan<- error) {
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

func (p *Peer) notifyProtocols() {
	p.wg.Add(len(p.protocolMap))
	for _, proto := range p.protocolMap {
		go func() {
			defer p.wg.Done()

			if proto.AddPeer != nil {
				proto.AddPeer(p, &proto)
			}
		}()
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
		return fmt.Errorf(fmt.Sprintf("could not found mapping proto with code %d", msgRecv.Code))
	}

	protocolTarget.in <- msgRecv

	return nil
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
	select {
	case msg := <-rw.in:
		msg.Code -= rw.offset

		return msg, nil
	}
}
