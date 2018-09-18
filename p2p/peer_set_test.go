/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/stretchr/testify/assert"
)

func getPeer() *Peer {
	node := discovery.NewNode(*crypto.MustGenerateRandomAddress(), nil, 0, 1)
	return NewPeer(nil, nil, node)
}

func Test_PeerSet(t *testing.T) {
	set := NewPeerSet()

	p1 := getPeer()
	set.add(p1)

	id, err := common.NewAddress(p1.Node.ID.Bytes())
	assert.Equal(t, err, nil)

	p2 := set.find(id)
	assert.Equal(t, p1, p2)
	assert.Equal(t, set.count(), 1)

	id1 := *crypto.MustGenerateRandomAddress()
	p3 := set.find(id1)
	assert.Equal(t, p3, (*Peer)(nil))

	set.delete(p1)
	assert.Equal(t, set.count(), 0)
}
