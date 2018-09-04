/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

import (
	"math/big"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/stretchr/testify/assert"
)

type TestDownloadPeer struct{}

func (s TestDownloadPeer) Head() (common.Hash, *big.Int) {
	return common.EmptyHash, nil
}
func (s TestDownloadPeer) RequestHeadersByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int, reverse bool) error {
	return nil
}
func (s TestDownloadPeer) RequestBlocksByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int) error {
	return nil
}

func Test_Download_NewPeerConnAndClose(t *testing.T) {
	var peer TestDownloadPeer
	peerID := "testPeerID"

	pc := newPeerConn(peer, peerID, nil)
	defer pc.close()

	assert.Equal(t, pc != nil, true)
	assert.Equal(t, pc.peerID, peerID)
	assert.Equal(t, pc.peer, peer)
}

func Test_Download_WaitMsg(t *testing.T) {
	magic := uint32(1)
	msgCode := uint16(0)
	cancelCh := make(chan struct{})

	// quit message
	pc := testPeerConn()
	go func() {
		_, err := pc.waitMsg(magic, msgCode, cancelCh)
		assert.Equal(t, err, errPeerQuit)
	}()
	time.Sleep(100 * time.Millisecond)
	pc.close()

	// cancel message
	pc = testPeerConn()
	go func() {
		_, err := pc.waitMsg(magic, msgCode, cancelCh)
		assert.Equal(t, err, errReceivedQuitMsg)
	}()
	time.Sleep(100 * time.Millisecond)
	close(cancelCh)

	// BlockHeadersMsg
	msgCode = BlockHeadersMsg
	cancelCh = make(chan struct{})
	msg := newMessage(BlockHeadersMsg)
	pc = testPeerConn()

	go func() {
		ret, err := pc.waitMsg(magic, msgCode, cancelCh)
		assert.Equal(t, err, nil)
		assert.Equal(t, ret != nil, true)
	}()
	time.Sleep(100 * time.Millisecond)
	pc.waitingMsgMap[BlockHeadersMsg] <- msg

	// BlocksMsg
	msgCode = BlocksMsg
	cancelCh = make(chan struct{})
	msg = newMessage(BlocksMsg)
	pc = testPeerConn()

	go func() {
		ret, err := pc.waitMsg(magic, msgCode, cancelCh)
		assert.Equal(t, err, nil)
		assert.Equal(t, ret != nil, true)
	}()
	time.Sleep(100 * time.Millisecond)
	pc.waitingMsgMap[BlocksMsg] <- msg

	// BlocksMsg sent by deliverMsg
	msgCode = BlocksMsg
	cancelCh = make(chan struct{})
	msg = newMessage(BlocksMsg)
	pc = testPeerConn()

	go func() {
		ret, err := pc.waitMsg(magic, msgCode, cancelCh)
		assert.Equal(t, err, nil)
		assert.Equal(t, ret != nil, true)
	}()
	time.Sleep(100 * time.Millisecond)
	pc.deliverMsg(msgCode, msg)
}

func testPeerConn() *peerConn {
	var peer TestDownloadPeer
	peerID := "testPeerID"
	pc := newPeerConn(peer, peerID, nil)

	return pc
}

const (
	ctlMsgProtoHandshake uint16 = 10
	ctlMsgDiscCode       uint16 = 4
	ctlMsgPingCode       uint16 = 3
	ctlMsgPongCode       uint16 = 4
)

func newMessage(code uint16) *p2p.Message {
	return &p2p.Message{
		Code:    code,
		Payload: []byte("payLoad"),
	}
}
