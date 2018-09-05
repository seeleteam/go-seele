/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/seeleteam/go-seele/common"
)

func Test_Server_StartService(t *testing.T) {
	nodeDir := "."
	myID := common.HexMustToAddres("0xd0c549b022f5a17a8f50a4a448d20ba579d01781")
	myAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9777")
	bootstrap := make([]*Node, 0)
	shard := uint(1)

	db := StartService(nodeDir, myID, myAddr, bootstrap, shard)
	assert.Equal(t, db != nil, true)
}
