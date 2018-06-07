/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

func getPeer() *Peer {
	node := discovery.NewNode(*crypto.MustGenerateRandomAddress(), nil, 0, 1)
	return NewPeer(nil, nil, nil, node)
}

func Test_PeerSet(t *testing.T) {
	set := NewPeerSet()

	p1 := getPeer()
	set.add(p1)

	id, err := common.NewAddress(p1.Node.ID.Bytes())
	assert.Equal(t, err, nil)

	p2 := set.find(id)
	assert.Equal(t, p1, p2)

	set.delete(p1)
	assert.Equal(t, set.count(), 0)
}
