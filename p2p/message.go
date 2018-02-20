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
	MsgCode    uint16 // message code, defined in each protocol
	Payload    []byte
	ReceivedAt time.Time
	CurPeer    *Peer // peer that handle this message
}

// msg wrapped Message, used in p2p layer
type msg struct {
	protoCode uint16
	msgCode   uint16
	size      uint32
	payload   []byte
}

// ProtoHandShake handshake message for two peer to exchage base information
// TODO add public key or other information for encryption?
type ProtoHandShake struct {
	Caps   []Cap
	NodeID discovery.NodeID
	Nounce uint32 //
}
