/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"net"
	"time"
)

// connection TODO add bandwidth meter for connection
type connection struct {
	fd net.Conn // tcp connection
	//node *discovery.Node // remote peer that this peer connects
}

// readFull receive from fd till outBuf is full
func (p *connection) readFull(outBuf []byte) (err error) {
	needLen, curPos := len(outBuf), 0
	p.fd.SetReadDeadline(time.Now().Add(frameReadTimeout))
	for needLen > 0 && err == nil {
		var nRead int
		nRead, err = p.fd.Read(outBuf[curPos:])
		needLen -= nRead
		curPos += nRead
	}
	if err != nil {
		// discard the input data
		return err
	}
	return nil
}
