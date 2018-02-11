/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package utils

import (
	"encoding/hex"
	"github.com/magiconair/properties/assert"
	"testing"
)

func Test_NodeId(t *testing.T) {
	id := "snode://c03ff3c956d0a320b153a097c3d04efa488d43d7d7e05a44791492c9979ff558f9956c0a6b0c414783476f02ad8557349d35ba9373dadfa9a7a44fd88328189f@192.168.122.132:9000"

	node, err := NewNodeId(id)
	assert.Equal(t, err, nil)

	hex := hex.EncodeToString(node.Address.Bytes())
	assert.Equal(t, hex, "c03ff3c956d0a320b153a097c3d04efa488d43d7d7e05a44791492c9979ff558f9956c0a6b0c414783476f02ad8557349d35ba9373dadfa9a7a44fd88328189f")
	assert.Equal(t, node.IP.String(), "192.168.122.132")
	assert.Equal(t, node.Port, 9000)

	assert.Equal(t, node.String(), id)
}