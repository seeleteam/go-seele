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
	// magic used to check the data head
	magic       = [2]byte{'^', '~'}
	maxSize     = uint32(8 * 1024 * 1024)
	magicNumber = binary.BigEndian.Uint16(magic[:])
)

var (
	errConnWriteTimeout = errors.New("Connection writes timeout")
	errMagic            = errors.New("Failed to wait magic")
	errSize             = errors.New("Failed to get data, size is too big")
)

// connection
// TODO add bandwidth metrics
type connection struct {
	// tcp connection
	fd net.Conn

	// read msg lock
	rmutux sync.Mutex

	// write msg lock
	wmutux sync.Mutex

	// writeErr if error appeared, tcp connection needs to be closed
	writeErr error

	// log
	log *log.SeeleLog
}

// readFull receive from fd till outBuf is full,
// if no data is read (with deadline of frameReadTimeout), returns timeout.
func (c *connection) readFull(outBuf []byte) (err error) {
	return c.readFullTimeout(outBuf, frameReadTimeout)
}

func (c *connection) readFullTimeout(outBuf []byte, timeout time.Duration) (err error) {
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
	return c.writeFullTimeout(outBuf, connWriteTimeout)
}

func (c *connection) writeFullTimeout(outBuf []byte, timeout time.Duration) (err error) {
	needLen, curPos := len(outBuf), 0
	for needLen > 0 {
		err = c.fd.SetWriteDeadline(time.Now().Add(timeout))
		if err != nil {
			// set writeErr with err, tcp connection will be closed when read again
			c.writeErr = err

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

	if err != nil {
		// set writeErr with err, tcp connection will be closed when read again
		c.writeErr = err
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

	if c.writeErr != nil {
		c.log.Debug("conn ReadMsg writeErr not nil. sender is %s, err= %s. ", c.fd.RemoteAddr().String(), c.writeErr)
		err = c.writeErr
		c.writeErr = nil
		return &Message{}, err
	}

	headbuff := make([]byte, headBuffLength)
	if err = c.readFull(headbuff); err != nil {

		return &Message{}, err
	}

	msgRecv = &Message{
		Code: binary.BigEndian.Uint16(headbuff[headBuffCodeStart:headBuffCodeEnd]),
	}

	size := binary.BigEndian.Uint32(headbuff[headBuffSizeStart:headBuffSizeEnd])
	receive := binary.BigEndian.Uint16(headbuff[headBuffMagicStart:headBuffMagicEnd])
	if magicNumber != receive {
		c.log.Debug("Failed to wait magic %d, got %d, sender is %s", magicNumber, receive, c.fd.RemoteAddr().String())

		return &Message{}, errMagic
	}

	if size > maxSize {
		c.log.Debug("Failed to get data, payload size %d exceeds the limit %d, sender is %s", size, maxSize, c.fd.RemoteAddr().String())

		return &Message{}, errSize
	}

	if size > 0 {
		msgRecv.Payload = make([]byte, size)
		if err = c.readFull(msgRecv.Payload); err != nil {

			return &Message{}, err
		}

		/*todo disable zip
		if err = msgRecv.UnZip(); err != nil {
			return &Message{}, err
		}*/
	}

	metricsReceiveMessageCountMeter.Mark(1)
	metricsReceivePortSpeedMeter.Mark(headBuffLength + int64(size))

	return msgRecv, nil
}

// WriteMsg message can be any data type
func (c *connection) WriteMsg(msg *Message) error {
	c.wmutux.Lock()
	defer c.wmutux.Unlock()

	/*	todo disable zip
		if err := msg.Zip(); err != nil {
				return err
			}
	*/

	b := make([]byte, headBuffLength)
	binary.BigEndian.PutUint32(b[headBuffSizeStart:headBuffSizeEnd], uint32(len(msg.Payload)))
	binary.BigEndian.PutUint16(b[headBuffCodeStart:headBuffCodeEnd], msg.Code)
	binary.BigEndian.PutUint16(b[headBuffMagicStart:headBuffMagicEnd], magicNumber)

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
