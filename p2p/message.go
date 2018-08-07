/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"time"

	"github.com/seeleteam/go-seele/common"
)

const (
	ctlMsgProtoHandshake uint16 = 10
	ctlMsgDiscCode       uint16 = 4
	ctlMsgPingCode       uint16 = 3
	ctlMsgPongCode       uint16 = 4
)

const zipBytesLimit = 1024 * 1024

// Message exposed for high level layer to receive
type Message struct {
	Code       uint16 // message code, defined in each protocol
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

// Zip compress message when the length of payload is greater than zipBytesLimit
func (m *Message) Zip() error {
	if len(m.Payload) <= zipBytesLimit {
		return nil
	}

	buf := new(bytes.Buffer)

	w := gzip.NewWriter(buf)
	if _, err := w.Write(m.Payload); err != nil {
		return err
	}
	w.Close()
	m.Payload = buf.Bytes()

	return nil
}

// UnZip regardless of whether the message is compressed to be decompress
func (m *Message) UnZip() error {
	if len(m.Payload) == 0 {
		return nil
	}

	pl := bytes.NewReader(m.Payload)
	r, err := gzip.NewReader(pl)
	if err == gzip.ErrHeader || err == gzip.ErrChecksum {
		return nil
	}
	if err != nil {
		return err
	}
	defer r.Close()

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	m.Payload = b
	return nil
}

// ProtoHandShake handshake message for two peer to exchange base information
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
