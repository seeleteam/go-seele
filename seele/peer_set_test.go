/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"net"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

func getTestPeer() *peer {
	addr := crypto.MustGenerateRandomAddress()
	node := discovery.NewNodeWithAddr(*addr, &net.UDPAddr{}, 0)
	p2pPeer := p2p.NewPeer(nil, nil, nil, node)
	peer := newPeer(1, p2pPeer, nil)

	return peer
}

func Test_PeerSet_Add(t *testing.T) {
	set := newPeerSet()

	peer1 := getTestPeer()
	set.Add(peer1)
	assert.Equal(t, len(set.peers), 1)

	set.Add(peer1)
	assert.Equal(t, len(set.peers), 1)

	peer2 := getTestPeer()
	set.Add(peer2)
	assert.Equal(t, len(set.peers), 2)
}

func Test_PeerSet_Find(t *testing.T) {
	set := newPeerSet()
	peer1 := getTestPeer()
	set.Add(peer1)
	peer2 := getTestPeer()
	set.Add(peer2)

	assert.Equal(t, set.Find(peer1.Node.ID), peer1)
	assert.Equal(t, set.Find(peer2.Node.ID), peer2)
}

func TestPeerSet_ForEach(t *testing.T) {
	set := newPeerSet()
	peer1 := getTestPeer()
	set.Add(peer1)
	peer2 := getTestPeer()
	set.Add(peer2)

	count := 0
	set.ForEach(func(peer *peer) bool {
		count++
		return true
	})

	assert.Equal(t, count, 2)

	set.ForEach(func(peer *peer) bool {
		count++
		if count == 3 {
			return false
		}

		return true
	})
	assert.Equal(t, count, 3)
}

func Test_PeerSet_Remove(t *testing.T) {
	set := newPeerSet()
	peer1 := getTestPeer()
	set.Add(peer1)
	peer2 := getTestPeer()
	set.Add(peer2)

	assert.Equal(t, len(set.peers), 2)
	set.Remove(peer1.Node.ID)
	assert.Equal(t, len(set.peers), 1)
	set.Remove(peer1.Node.ID)
	assert.Equal(t, len(set.peers), 1)
	set.Remove(peer2.Node.ID)
	assert.Equal(t, len(set.peers), 0)

}
