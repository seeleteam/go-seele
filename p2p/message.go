/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"time"

	"github.com/seeleteam/go-seele/p2p/discovery"
)

const (
	ctlMsgProtoHandshake uint16 = 10
	ctlMsgDiscCode       uint16 = 4
	ctlMsgPingCode       uint16 = 3
	ctlMsgPongCode       uint16 = 4
)

// Message exposed for high level layer to receive
type Message struct {
	Code       uint16 // message code, defined in each protocol
	Payload    []byte
	ReceivedAt time.Time
}

// ProtoHandShake handshake message for two peer to exchage base information
// TODO add public key or other information for encryption?
type ProtoHandShake struct {
	Caps   []Cap
	NodeID discovery.NodeID
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
