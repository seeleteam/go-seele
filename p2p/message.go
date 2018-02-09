/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"time"
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
