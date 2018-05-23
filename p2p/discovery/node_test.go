/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"encoding/hex"
	"testing"

	"github.com/magiconair/properties/assert"
)

func Test_NodeId(t *testing.T) {
	id := "snode://c03ff3c956d0a320b153a097c3d04efa488d43d7d7e05a44791492c9979ff558f9956c0a6b0c414783476f02ad8557349d35ba9373dadfa9a7a44fd88328189f@192.168.122.132:9000[1]"

	node, err := NewNodeFromString(id)
	assert.Equal(t, err, nil)

	hex := hex.EncodeToString(node.ID.Bytes())
	assert.Equal(t, hex, "c03ff3c956d0a320b153a097c3d04efa488d43d7d7e05a44791492c9979ff558f9956c0a6b0c414783476f02ad8557349d35ba9373dadfa9a7a44fd88328189f")
	assert.Equal(t, node.IP.String(), "192.168.122.132")
	assert.Equal(t, node.UDPPort, 9000)
	assert.Equal(t, node.Shard, uint(1))

	assert.Equal(t, node.String(), id)
}
