/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"
	"testing"

	"github.com/seeleteam/go-seele/crypto"
	"github.com/stretchr/testify/assert"
)

func Test_Server_StartService(t *testing.T) {
	nodeDir := "."
	myID := *crypto.MustGenerateShardAddress(1)
	myAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9777")
	bootstrap := make([]*Node, 0)
	shard := uint(1)

	db := StartService(nodeDir, myID, myAddr, bootstrap, shard)
	assert.Equal(t, db != nil, true)
}
