/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"net"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/stretchr/testify/assert"
)

func newTestPeer(addr string, shard uint) (*Peer, error) {
	myAddr, err := common.HexToAddress(addr)
	if err != nil {
		return nil, err
	}

	node := discovery.NewNode(myAddr, nil, 0, shard)

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	defer ln.Close()

	c, err := net.Dial(ln.Addr().Network(), ln.Addr().String())
	if err != nil {
		return nil, err
	}

	newPeer := NewPeer(&connection{fd: c}, log.GetLogger("peer"), node)
	return newPeer, nil
}

func Test_peer_Info(t *testing.T) {
	addr := crypto.MustGenerateShardAddress(1).Hex()
	newPeer, err := newTestPeer(addr, 1)
	if err != nil {
		t.Fatal(err)
	}

	info := newPeer.Info()

	assert.Equal(t, info.Shard, uint(1))
	assert.Equal(t, info.ID, addr)
}

func Test_peer_RunAndClose(t *testing.T) {
	p1, err := newTestPeer(crypto.MustGenerateShardAddress(1).Hex(), 1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, p1 != nil, true)
	assert.Equal(t, p1.getShardNumber(), uint(1))

	p1.close()
	_, ok := <-p1.closed
	_, ok1 := <-p1.protocolErr

	assert.Equal(t, p1.disconnection, (chan string)(nil))
	assert.Equal(t, ok, false)
	assert.Equal(t, ok1, false)

	// wrong test, comment it out
	//p2, err := newTestPeer("0xc31b35a3600eb13ebbc9f504924e747d854c1421", 1)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//go func() {
	//	err := p2.run()
	//	assert.Nil(t, p2.disconnection)
	//	assert.NotNil(t, err)
	//	assert.Equal(t, strings.Contains(err.Error(), "123"), true)
	//}()
	//
	//p2.Disconnect("123")
	//assert.Nil(t, p2.disconnection)
	//p2.wg.Wait()
	//
	//_, ok2 := <-p2.closed
	//_, ok3 := <-p2.protocolErr
	//
	//assert.Equal(t, ok2, false)
	//assert.Equal(t, ok3, false)
}
