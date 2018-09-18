/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"testing"

	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func newTestBlockchainDatabase() (store.BlockchainStore, func()) {
	db, dispose := leveldb.NewTestDatabase()
	return store.NewBlockchainDatabase(db), dispose
}

func Test_LightChain_NewLightChain(t *testing.T) {
	set := newPeerSet()

	peer1 := getTestPeer(0)
	set.Add(peer1)
	assert.Equal(t, len(set.peerMap), 1)
	assert.Equal(t, len(set.shardPeers[0]), 1)

	set.Add(peer1)
	assert.Equal(t, len(set.peerMap), 1)
	assert.Equal(t, len(set.shardPeers[0]), 1)

	peer2 := getTestPeer(1)
	set.Add(peer2)
	assert.Equal(t, len(set.peerMap), 2)
	assert.Equal(t, len(set.shardPeers[1]), 1)
}
