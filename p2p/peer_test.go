/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"net"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

func Test_peer_Info(t *testing.T) {
	myAddr := common.HexMustToAddres("0x6b9fd39a9f1273c46fba8951b62de5b95cd3dd84")
	node := discovery.NewNode(myAddr, nil, 0, 1)

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	c, err := net.Dial(ln.Addr().Network(), ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	newPeer := NewPeer(&connection{fd: c}, nil, nil, node)
	info := newPeer.Info()

	assert.Equal(t, info.Shard, uint(1))
	assert.Equal(t, info.ID, myAddr.ToHex())
}
