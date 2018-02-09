/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"
	"testing"

	"github.com/magiconair/properties/assert"
)

func Test_Bucket(t *testing.T) {
	b := bucket{}

	n := getNode("9000")
	b.addNode(n)
	assert.Equal(t, b.size(), 1)
	assert.Equal(t, b.hasNode(n), 0)

	b.addNode(n)
	assert.Equal(t, b.size(), 1)

	// copy node of n
	n3 := NewNode(n.ID, n.IP, n.UDPPort)
	b.addNode(n3)
	assert.Equal(t, b.size(), 1)

	n2 := getNode("9001")
	b.addNode(n2)
	assert.Equal(t, b.size(), 2)

	b.deleteNode(n.getSha())
	assert.Equal(t, b.size(), 1)

	b.deleteNode(n2.getSha())
	assert.Equal(t, b.size(), 0)
}

func getNode(port string) *Node {
	id := getRandomNodeID()
	addr, _ := net.ResolveUDPAddr("udp", ":"+port)
	n := NewNode(id, addr.IP, uint16(addr.Port))

	return n
}
