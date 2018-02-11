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

// Message exposed for high level layer to call
type Message struct {
	msgCode    uint16 // message code, defined in each protocol
	size       uint32 // size of the paylod
	payload    []byte
	ReceivedAt time.Time
	CurPeer    *Peer // peer that handle this message
}

// msg wrapped Message, used in p2p layer
type msg struct {
	Message
	protoCode uint16
}

type protoHandShake struct {
	caps   []Cap
	nodeId discovery.NodeID
	nounce uint32 //
}
