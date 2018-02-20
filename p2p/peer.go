/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

const (
	pingInterval         = 3 * time.Second // ping interval for peer tcp connection. Should be 15
	discAlreadyConnected = 10              // node already has connection
	discServerQuit       = 11              // p2p.server need quit, all peers should quit as it can
)

// Peer represents a connected remote node.
type Peer struct {
	conn     *connection     // tcp connection
	node     *discovery.Node // remote peer that this peer connects
	created  uint64          // Peer create time, nanosecond
	err      error
	closed   chan struct{}
	disc     chan uint
	protoMap map[uint16]*Protocol // protoCode=>proto
	capMap   map[string]uint16    // cap of protocol => protoCode

	wMutex sync.Mutex // for conn write
	wg     sync.WaitGroup
	log    *log.SeeleLog
}

// run assumes that SubProtocol will never quit, otherwise proto.DelPeerCh may be closed before peer.run quits?
func (p *Peer) run() {
	// add peer to protocols
	var (
		writeErr = make(chan error, 1)
		readErr  = make(chan error, 1)
		err      error
	)
	for _, proto := range p.protoMap {
		proto.AddPeerCh <- p
	}

	p.wg.Add(2)
	go p.readLoop(readErr)
	go p.pingLoop()

	// Wait for an error or disconnect.
loop:
	for {
		select {
		case err = <-writeErr:
			// A write finished. Allow the next write to start if
			// there was no error.
			if err != nil {
				p.err = err
				break loop
			}
		case err = <-readErr:
			p.log.Info("p2p.peer.run read err? %s", err)
			p.err = err
			break loop
		case <-p.disc:
			p.err = errors.New("disc error recved")
			break loop
		}
	}

	p.close()
	p.wg.Wait()
	// send delpeer message for each protocols
	for _, proto := range p.protoMap {
		proto.DelPeerCh <- p
	}
	p.log.Info("p2p.peer.run quit. err=%s", p.err)
}

func (p *Peer) close() {
	close(p.closed)
	p.conn.fd.Close()
	close(p.disc)
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
		msgRecv, err := p.recvRawMsg()
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

func (p *Peer) handle(msgRecv *msg) error {
	proto, ok := p.protoMap[msgRecv.protoCode]
	if ok {
		/*var dataMsg interface{}
		if msgRecv.payload != nil {
			if err := common.Deserialize(msgRecv.payload, dataMsg); err != nil {
				p.log.Info("peer.handle err, %s", err)
				return nil
			}
		}*/
		msgTo := &Message{
			MsgCode:    msgRecv.msgCode,
			ReceivedAt: time.Now(),
			CurPeer:    p,
			Payload:    msgRecv.payload,
		}
		select {
		case proto.ReadMsgCh <- msgTo:
			return nil
		case <-p.closed:
			return io.EOF
		}
	}

	if msgRecv.protoCode != 1 {
		return errors.New("not valid protoCode")
	}
	// for control msg
	switch {
	case msgRecv.msgCode == ctlMsgPingCode:
		go p.sendCtlMsg(ctlMsgPongCode)
	case msgRecv.msgCode == ctlMsgDiscCode:
		return fmt.Errorf("error=%d", ctlMsgDiscCode)
	}
	return nil
}

// SendMsg called by subProtocols, message can be any data type
func (p *Peer) SendMsg(proto *Protocol, msgCode uint16, message interface{}) error {
	protoCode, ok := p.capMap[proto.cap().String()]
	if !ok {
		return errors.New("Not Found protoCode")
	}
	//fmt.Println("SendMsg called", msgCode)
	payload, err := common.Serialize(message)
	if err != nil {
		return errors.New("SendMsg yet message is not valid")
	}
	msgRaw := &msg{
		protoCode: protoCode,
		msgCode:   msgCode,
		size:      uint32(len(payload)),
		payload:   payload,
	}
	return p.sendRawMsg(msgRaw)
}

func (p *Peer) sendCtlMsg(msgCode uint16) error {
	hsMsg := &msg{
		protoCode: ctlProtoCode,
		msgCode:   msgCode,
	}
	hsMsg.size = 0
	p.sendRawMsg(hsMsg)
	return nil
}

func (p *Peer) sendRawMsg(msgSend *msg) error {
	p.wMutex.Lock()
	defer p.wMutex.Unlock()
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b[:4], msgSend.size)
	binary.BigEndian.PutUint16(b[4:6], msgSend.protoCode)
	binary.BigEndian.PutUint16(b[6:8], msgSend.msgCode)
	p.conn.fd.SetWriteDeadline(time.Now().Add(frameWriteTimeout))

	_, err := p.conn.fd.Write(b)
	if err != nil {
		return err
	}

	if msgSend.size > 0 {
		_, err = p.conn.fd.Write(msgSend.payload)
		if err != nil {
			return err
		}
	}
	p.log.Debug("sendRawMsg protoCode:%d msgCode:%d", msgSend.protoCode, msgSend.msgCode)
	return nil
}

func (p *Peer) recvRawMsg() (msgRecv *msg, err error) {
	headbuf := make([]byte, 8)
	p.conn.fd.SetReadDeadline(time.Now().Add(frameReadTimeout))
	_, err1 := io.ReadFull(p.conn.fd, headbuf)

	if err1 != nil {
		return nil, err1
	}
	msgRecv = &msg{
		protoCode: binary.BigEndian.Uint16(headbuf[4:6]),
		size:      binary.BigEndian.Uint32(headbuf[:4]),
		msgCode:   binary.BigEndian.Uint16(headbuf[6:8]),
	}
	if msgRecv.size > 0 {
		msgRecv.payload = make([]byte, msgRecv.size)
		if _, err := io.ReadFull(p.conn.fd, msgRecv.payload); err != nil {
			return nil, err
		}
	}
	//	msgRecv.ReceivedAt = time.Now()
	//	msgRecv.CurPeer = p
	p.log.Debug("recvRawMsg protoCode:%d msgCode:%d", msgRecv.protoCode, msgRecv.msgCode)
	return msgRecv, nil
}

// Disconnect terminates the peer connection with the given reason.
// It returns immediately and does not wait until the connection is closed.
func (p *Peer) Disconnect(reason uint) {
	select {
	case p.disc <- reason:
	case <-p.closed:
	}
}
