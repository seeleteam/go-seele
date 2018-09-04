/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/magiconair/properties/assert"
)

func Test_Net_GetUDPConn(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:9898")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := getUDPConn(addr)
	defer conn.Close()

	fmt.Println("getUDPConn 0 err:", err)
	assert.Equal(t, err, nil)
	fmt.Println("getUDPConn 0 conn:", conn)
	assert.Equal(t, conn != nil, true)

	// failed to listen due to already binded
	conn, err = getUDPConn(addr)
	fmt.Println("getUDPConn 1 err:", err)
	assert.Equal(t, err != nil, true)
	assert.Equal(t, strings.Contains(err.Error(), "bind: Only one usage of each socket address"), true)
	fmt.Println("getUDPConn 1 conn:", conn)
	assert.Equal(t, conn == nil, true)
}
