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
	headBuffSizeStart  = 0
	headBuffSizeEnd    = 4
	headBuffCodeStart  = 4
	headBuffCodeEnd    = 6
	headBuffMagicStart = 6
	headBuffMagicEnd   = 8
	maxSize            = 8 * 1024 * 1024
)

var (
	// magic used to check the data head
	magic = [2]byte{'s', 'l'}
)

var (
	errConnWriteTimeout = errors.New("Connection writes timeout")
)

// connection
// TODO add bandwidth metrics
type connection struct {
	fd     net.Conn   // tcp connection
	rmutux sync.Mutex // read msg lock
	wmutux sync.Mutex // write msg lock
}

// readFull receive from fd till outBuf is full,
// if no data is read (with deadline of frameReadTimeout), returns timeout.
func (c *connection) readFull(outBuf []byte) (err error) {
	return c.readFullo(outBuf, frameReadTimeout)
}

func (c *connection) readFullo(outBuf []byte, timeout time.Duration) (err error) {
	needLen, curPos := len(outBuf), 0

	err = c.fd.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return err
	}

	for needLen > 0 && err == nil {
		var nRead int
		nRead, err = c.fd.Read(outBuf[curPos:])
		needLen -= nRead
		curPos += nRead
	}

	return err
}

// writeFull write to fd till all outBuf is sended,
// if no data is writed (with deadline of connWriteTimeout), returns errConnWriteTimeout.
func (c *connection) writeFull(outBuf []byte) (err error) {
	return c.writeFullo(outBuf, connWriteTimeout)
}

func (c *connection) writeFullo(outBuf []byte, timeout time.Duration) (err error) {
	needLen, curPos := len(outBuf), 0
	for needLen > 0 {
		err = c.fd.SetWriteDeadline(time.Now().Add(timeout))
		if err != nil {
			return err
		}

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
func (c *connection) ReadMsg() (msgRecv *Message, err error) {
	c.rmutux.Lock()
	defer c.rmutux.Unlock()

	headbuff := make([]byte, headBuffLength)
	if err = c.readFull(headbuff); err != nil {
		return &Message{}, err
	}

	msgRecv = &Message{
		Code: binary.BigEndian.Uint16(headbuff[headBuffCodeStart:headBuffCodeEnd]),
	}

	size := binary.BigEndian.Uint32(headbuff[headBuffSizeStart:headBuffSizeEnd])
	magicNum := binary.BigEndian.Uint16(headbuff[headBuffMagicStart:headBuffMagicEnd])
	if binary.BigEndian.Uint16(magic[:]) != magicNum {
		mlog := log.GetLogger("p2p")
		mlog.Debug("Failed to wait magic %d, got %d, sender is %s", binary.BigEndian.Uint16(magic[:]), magicNum, c.fd.RemoteAddr().String())
		return &Message{}, errors.New("Failed to wait magic")
	}

	if size > maxSize {
		mlog := log.GetLogger("p2p")
		mlog.Debug("Failed to get data, size %d is so big, sender is %s", size, c.fd.RemoteAddr().String())
		return &Message{}, errors.New("Failed to get data, size is too big")
	}

	if size > 0 {
		msgRecv.Payload = make([]byte, size)
		if err = c.readFull(msgRecv.Payload); err != nil {
			return &Message{}, err
		}

		if err = msgRecv.UnZip(); err != nil {
			return &Message{}, err
		}
	}
	metricsReceiveMessageCountMeter.Mark(1)
	metricsReceivePortSpeedMeter.Mark(headBuffLength + int64(size))
	return msgRecv, nil
}

// WriteMsg message can be any data type
func (c *connection) WriteMsg(msg *Message) error {
	c.wmutux.Lock()
	defer c.wmutux.Unlock()

	if err := msg.Zip(); err != nil {
		return err
	}

	b := make([]byte, headBuffLength)
	binary.BigEndian.PutUint32(b[headBuffSizeStart:headBuffSizeEnd], uint32(len(msg.Payload)))
	binary.BigEndian.PutUint16(b[headBuffCodeStart:headBuffCodeEnd], msg.Code)
	binary.BigEndian.PutUint16(b[headBuffMagicStart:headBuffMagicEnd], binary.BigEndian.Uint16(magic[:]))

	if err := c.writeFull(b); err != nil {
		return err
	}

	if len(msg.Payload) > 0 {
		if err := c.writeFull(msg.Payload); err != nil {
			return err
		}
	}
	metricsSendMessageCountMeter.Mark(1)
	metricsSendPortSpeedMeter.Mark(headBuffLength + int64(len(msg.Payload)))
	return nil
}
