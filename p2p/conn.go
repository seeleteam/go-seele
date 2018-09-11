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

	"github.com/seeleteam/go-seele/log"
)

const (
	headBuffLength     = 8
	headBuffMagicStart = 0
	headBuffMagicEnd   = 2
	headBuffSizeStart  = 2
	headBuffSizeEnd    = 6
	headBuffCodeStart  = 6
	headBuffCodeEnd    = 8
)

var (
	errConnWriteTimeout = errors.New("Connection writes timeout")
	errMagic            = errors.New("Failed to wait magic")
	errSize             = errors.New("Failed to get data, size is too big")
)

// connection TODO add bandwidth meter for connection
type connection struct {
	fd     net.Conn   // tcp connection
	rmutux sync.Mutex // read msg lock
	wmutux sync.Mutex // write msg lock
}

var (
	// magic used to check the data head
	magic       = [2]byte{'^', '~'}
	maxSize     = uint32(8 * 1024 * 1024)
	magicNumber = binary.BigEndian.Uint16(magic[:])
)

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

	headbuff := make([]byte, headBuffLength)
	if err = c.readFull(headbuff); err != nil {
		return Message{}, err
	}

	msgRecv = Message{
		Code: binary.BigEndian.Uint16(headbuff[headBuffCodeStart:headBuffCodeEnd]),
	}

	size := binary.BigEndian.Uint32(headbuff[headBuffSizeStart:headBuffSizeEnd])
	receive := binary.BigEndian.Uint16(headbuff[headBuffMagicStart:headBuffMagicEnd])
	if magicNumber != receive {
		mlog := log.GetLogger("p2p", false)
		mlog.Debug("Failed to wait magic %d, got %d, sender is %s", magicNumber, receive, c.fd.RemoteAddr().String())
		return Message{}, errMagic
	}

	if size > maxSize {
		mlog := log.GetLogger("p2p", false)
		mlog.Debug("Failed to get data, payload size %d exceeds the limit %d, sender is %s", size, maxSize, c.fd.RemoteAddr().String())
		return Message{}, errSize
	}

	if size > 0 {
		msgRecv.Payload = make([]byte, size)
		if err = c.readFull(msgRecv.Payload); err != nil {
			return Message{}, err
		}
	}
	metricsReceiveMessageCountMeter.Mark(1)
	metricsReceivePortSpeedMeter.Mark(headBuffLength + int64(size))
	return msgRecv, nil
}

// WriteMsg message can be any data type
func (c *connection) WriteMsg(msg Message) error {
	c.wmutux.Lock()
	defer c.wmutux.Unlock()

	b := make([]byte, headBuffLength)
	binary.BigEndian.PutUint32(b[headBuffSizeStart:headBuffSizeEnd], uint32(len(msg.Payload)))
	binary.BigEndian.PutUint16(b[headBuffCodeStart:headBuffCodeEnd], msg.Code)
	binary.BigEndian.PutUint16(b[headBuffMagicStart:headBuffMagicEnd], magicNumber)
	err := c.writeFull(b)
	if err != nil {
		return err
	}

	if len(msg.Payload) > 0 {
		err = c.writeFull(msg.Payload)
		if err != nil {
			return err
		}
	}
	metricsSendMessageCountMeter.Mark(1)
	metricsSendPortSpeedMeter.Mark(headBuffLength + int64(len(msg.Payload)))
	return nil
}
