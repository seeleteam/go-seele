/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Net_GetUDPConn(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:9898")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := getUDPConn(addr)
	defer conn.Close()

	assert.Equal(t, err, nil)
	assert.Equal(t, conn != nil, true)

	// failed to listen due to already binded
	conn, err = getUDPConn(addr)
	assert.Equal(t, err != nil, true)
	assert.Equal(t, strings.Contains(err.Error(), "bind:"), true)
	assert.Equal(t, conn == nil, true)
}
