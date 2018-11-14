/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"net"
	"testing"

	"github.com/seeleteam/go-seele/crypto"
	log2 "github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/stretchr/testify/assert"
)

func getTestPeer(shard uint) *peer {
	log := log2.GetLogger("test")
	addr := crypto.MustGenerateRandomAddress()
	node := discovery.NewNodeWithAddr(*addr, &net.UDPAddr{}, shard)
	p2pPeer := p2p.NewPeer(nil, nil, node)
	peer := newPeer(1, p2pPeer, nil, log, nil)

	return peer
}

func Test_PeerSet_Add(t *testing.T) {
	set := newPeerSet()

	peer1 := getTestPeer(0)
	set.Add(peer1)
	assert.Equal(t, len(set.peerMap), 1)

	set.Add(peer1)
	assert.Equal(t, len(set.peerMap), 1)

	peer2 := getTestPeer(1)
	set.Add(peer2)
	assert.Equal(t, len(set.peerMap), 2)
}

func Test_PeerSet_Find(t *testing.T) {
	set := newPeerSet()
	peer1 := getTestPeer(0)
	set.Add(peer1)
	peer2 := getTestPeer(0)
	set.Add(peer2)

	assert.Equal(t, set.Find(peer1.Node.ID), peer1)
	assert.Equal(t, set.Find(peer2.Node.ID), peer2)
}

func Test_PeerSet_Remove(t *testing.T) {
	set := newPeerSet()
	peer1 := getTestPeer(0)
	set.Add(peer1)
	peer2 := getTestPeer(1)
	set.Add(peer2)

	assert.Equal(t, len(set.peerMap), 2)
	set.Remove(peer1.Node.ID)
	assert.Equal(t, len(set.peerMap), 1)
	set.Remove(peer1.Node.ID)
	assert.Equal(t, len(set.peerMap), 1)
	set.Remove(peer2.Node.ID)
	assert.Equal(t, len(set.peerMap), 0)
}
