/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"time"

	"github.com/seeleteam/go-seele/common"
)

const (
	ctlMsgProtoHandshake uint16 = 10
	ctlMsgDiscCode       uint16 = 4
	ctlMsgPingCode       uint16 = 3
	ctlMsgPongCode       uint16 = 4
	ctlMsgZipCode        uint16 = 5
)

const zipLimit int = 1024

// Message exposed for high level layer to receive
type Message struct {
	Code       uint16 // message code, defined in each protocol
	ZipCode    uint16
	Payload    []byte
	ReceivedAt time.Time
}

func SendMessage(write MsgWriter, code uint16, payload []byte) error {
	msg := Message{
		Code:    code,
		Payload: payload,
	}

	return write.WriteMsg(msg)
}

// ZipMessage zip message when the length of payload is greater than zipLimit
func (m *Message) ZipMessage() error {
	if len(m.Payload) > zipLimit {
		buf := new(bytes.Buffer)

		w := gzip.NewWriter(buf)
		defer w.Close()
		_, err := w.Write(m.Payload)
		if err != nil {
			return err
		}
		err = w.Flush()
		if err != nil {
			return err
		}
		m.Payload = buf.Bytes()
		m.ZipCode = ctlMsgZipCode
	}
	return nil
}

// UZipMessage uzip message when m.ZipCode equal ctlMsgZipCode
func (m *Message) UZipMessage() error {
	if m.ZipCode != ctlMsgZipCode {
		return nil
	}
	buf := new(bytes.Buffer)
	_, err := buf.Write(m.Payload)
	if err != nil {
		return err
	}
	r, err := gzip.NewReader(buf)
	defer r.Close()
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(r)
	if err != nil && err != io.ErrUnexpectedEOF {
		return err
	}
	m.Payload = b
	return nil
}

// ProtoHandShake handshake message for two peer to exchage base information
// TODO add public key or other information for encryption?
type ProtoHandShake struct {
	Caps      []Cap
	NodeID    common.Address
	Params    []byte
	NetworkID uint64
}

type MsgReader interface {
	// ReadMsg read a message. It will block until send the message out or get errors
	ReadMsg() (Message, error)
}

type MsgWriter interface {
	// WriteMsg sends a message. It will block until the message's
	// Payload has been consumed by the other end.
	//
	// Note that messages can be sent only once because their
	// payload reader is drained.
	WriteMsg(Message) error
}

// MsgReadWriter provides reading and writing of encoded messages.
// Implementations should ensure that ReadMsg and WriteMsg can be
// called simultaneously from multiple goroutines.
type MsgReadWriter interface {
	MsgReader
	MsgWriter
}
