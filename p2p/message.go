/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"time"
)

// Message
type Message struct {
	msgCode    uint16
	size       uint32 // size of the paylod
	payload    []byte
	ReceivedAt time.Time
	CurPeer    *Peer
}

// msg
type msg struct {
	Message
	protoCode uint16
}
