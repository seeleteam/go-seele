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
	"net"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/p2p/discovery"
)

type Peer struct {
	fd       net.Conn
	node     *discovery.Node
	created  uint64
	err      error
	closed   chan struct{}
	disc     chan uint
	protoMap map[uint16]*Protocol // protoCode=>proto
	capMap   map[string]uint16    //cap of protocol => protoCode

	wMutex sync.Mutex // for conn write
	wg     sync.WaitGroup
}

func (p *Peer) run() {
	// add peer to protocols
	for _, proto := range p.protoMap {
		proto.AddPeerCh <- p
	}
	var (
		writeErr = make(chan error, 1)
		readErr  = make(chan error, 1)
		//reason   uint // sent to the peer
		err error
	)
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
				//reason = DiscNetworkError
				break loop
			}
		case err = <-readErr:
			p.err = err
			break loop
		//case err = <-p.disc:
		case <-p.disc:
			break loop
		}
	}

	close(p.closed)
	p.fd.Close()
	p.wg.Wait()
	// del peer from protocols
	for _, proto := range p.protoMap {
		proto.DelPeerCh <- p
	}
}

func (p *Peer) pingLoop() {
	ping := time.NewTimer(pingInterval)
	defer p.wg.Done()
	defer ping.Stop()
	for {
		select {
		case <-ping.C:
			p.sendCtlMsg(3)
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
		select {
		case proto.ReadMsgCh <- &(msgRecv.Message):
			return nil
		case <-p.closed:
			return io.EOF
		}
		return nil
	}

	if msgRecv.protoCode != 1 {
		return errors.New("not valid protoCode")
	}
	// for control msg
	switch {
	case msgRecv.msgCode == ctlMsgPingCode:
		go p.sendCtlMsg(ctlMsgPongCode)
	case msgRecv.msgCode == ctlMsgDiscCode:
		//var reason [1]DiscReason

		//rlp.Decode(msg.Payload, &reason)
		return fmt.Errorf("error=%d", ctlMsgDiscCode)
	}
	return nil
}

// SendMsg called by protocols
func (p *Peer) SendMsg(proto *Protocol, msgSend *Message) error {
	protoCode, ok := p.capMap[proto.cap().String()]
	if !ok {
		return errors.New("Not Found protoCode")
	}
	msgRaw := &msg{
		protoCode: protoCode,
		Message:   *msgSend,
	}
	return p.sendRawMsg(msgRaw)
}

func (p *Peer) sendCtlMsg(msgCode uint16) error {
	hsMsg := &msg{
		protoCode: 1,
		Message: Message{
			msgCode: msgCode,
		},
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
	p.fd.SetWriteDeadline(time.Now().Add(frameWriteTimeout))

	_, err := p.fd.Write(b)
	//fmt.Println("sendRawMsg,head", sendLen)
	if err != nil {
		return err
	}
	_, err = p.fd.Write(msgSend.payload)
	if err != nil {
		return err
	}
	fmt.Printf("sendRawMsg protoCode:%d msgCode:%d \r\n", msgSend.protoCode, msgSend.msgCode)
	return nil
}

func (p *Peer) recvRawMsg() (msgRecv *msg, err error) {
	headbuf := make([]byte, 8)
	p.fd.SetReadDeadline(time.Now().Add(frameReadTimeout))
	//readLen, err := p.fd.Read(headbuf)
	_, err1 := io.ReadFull(p.fd, headbuf)

	if err1 != nil {
		return nil, err1
	}
	//fmt.Println("recvRawMsg head", readLen)
	msgRecv = &msg{
		protoCode: binary.BigEndian.Uint16(headbuf[4:6]),
		Message: Message{
			size:    binary.BigEndian.Uint32(headbuf[:4]),
			msgCode: binary.BigEndian.Uint16(headbuf[6:8]),
		},
	}

	msgRecv.payload = make([]byte, msgRecv.size)
	if _, err := io.ReadFull(p.fd, msgRecv.payload); err != nil {
		return nil, err
	}
	msgRecv.ReceivedAt = time.Now()
	msgRecv.CurPeer = p
	fmt.Printf("recvRawMsg protoCode:%d msgCode:%d \r\n", msgRecv.protoCode, msgRecv.msgCode)
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
