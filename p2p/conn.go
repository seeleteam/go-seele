/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"
)

const (
	headBuffLegth     = 6
	headBuffSizeStart = 0
	headBuffSizeEnd   = 4
	headBuffCodeStart = 4
	headBuffCodeEnd   = 6
)

var (
	errConnWriteTimeout = errors.New("Connection writes timeout")
)

// connection TODO add bandwidth meter for connection
type connection struct {
	fd net.Conn // tcp connection

	rmutux sync.Mutex // read msg lock
	wmutux sync.Mutex // write msg lock
}

// readFull receive from fd till outBuf is full
func (c *connection) readFull(outBuf []byte) (err error) {
	needLen, curPos := len(outBuf), 0
	c.fd.SetReadDeadline(time.Now().Add(frameReadTimeout))
	for needLen > 0 && err == nil {
		var nRead int
		nRead, err = c.fd.Read(outBuf[curPos:])
		needLen -= nRead
		curPos += nRead
	}

	if err != nil {
		// discard the input data
		return err
	}

	return nil
}

// writeFull write to fd till all outBuf is sended,
// if no data is writed (with deadline of connWriteTimeout), returns errConnWriteTimeout.
func (c *connection) writeFull(outBuf []byte) (err error) {
	needLen, curPos := len(outBuf), 0
	for needLen > 0 {
		c.fd.SetWriteDeadline(time.Now().Add(connWriteTimeout))
		var curSend int
		curSend, err = c.fd.Write(outBuf[curPos:])
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				if curSend == 0 {
					err = errConnWriteTimeout
					break
				}
				needLen -= curSend
				curPos += curSend
				continue
			}
			break
		}
		needLen -= curSend
		curPos += curSend
	}

	return err
}

func (c *connection) close() {
	c.fd.Close()
}

// ReadMsg read msg with a full Message block
func (c *connection) ReadMsg() (msgRecv Message, err error) {
	c.rmutux.Lock()
	defer c.rmutux.Unlock()

	headbuff := make([]byte, headBuffLegth)
	if err = c.readFull(headbuff); err != nil {
		return Message{}, err
	}

	msgRecv = Message{
		Code: binary.BigEndian.Uint16(headbuff[headBuffCodeStart:headBuffCodeEnd]),
	}

	size := binary.BigEndian.Uint32(headbuff[headBuffSizeStart:headBuffSizeEnd])
	if size > 0 {
		msgRecv.Payload = make([]byte, size)
		if err = c.readFull(msgRecv.Payload); err != nil {
			return Message{}, err
		}
	}

	return msgRecv, nil
}

// WriteMsg message can be any data type
func (c *connection) WriteMsg(msg Message) error {
	c.wmutux.Lock()
	defer c.wmutux.Unlock()

	b := make([]byte, headBuffLegth)
	binary.BigEndian.PutUint32(b[headBuffSizeStart:headBuffSizeEnd], uint32(len(msg.Payload)))
	binary.BigEndian.PutUint16(b[headBuffCodeStart:headBuffCodeEnd], msg.Code)

	err := c.writeFull(b)
	if err != nil {
		return err
	}

	if len(msg.Payload) > 0 {
		_, err = c.fd.Write(msg.Payload)
		if err != nil {
			return err
		}
	}

	return nil
}
