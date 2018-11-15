/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NodeId(t *testing.T) {
	id := "snode://c3d04efa488d43d7d7e05a44791492c9979ff551@192.168.122.132:9000[1]"

	node, err := NewNodeFromString(id)
	assert.Equal(t, err, nil)

	hex := hex.EncodeToString(node.ID.Bytes())
	assert.Equal(t, hex, "c3d04efa488d43d7d7e05a44791492c9979ff551")
	assert.Equal(t, node.IP.String(), "192.168.122.132")
	assert.Equal(t, node.UDPPort, 9000)
	assert.Equal(t, node.Shard, uint(1))

	assert.Equal(t, node.String(), id)
}
