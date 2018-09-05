/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	log2 "github.com/seeleteam/go-seele/log"
)

func getBuckets() *bucket {
	log := log2.GetLogger("test")
	return newBuckets(log)
}

func Test_Bucket(t *testing.T) {
	b := getBuckets()

	n := getNode("9000")
	b.addNode(n)
	assert.Equal(t, b.size(), 1)
	assert.Equal(t, b.findNode(n), 0)

	b.addNode(n)
	assert.Equal(t, b.size(), 1)

	// copy node of n
	n3 := NewNode(n.ID, n.IP, int(n.UDPPort), 0)
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

func Test_AddNode(t *testing.T) {
	b := getBuckets()

	var n1 *Node
	for i := 0; i < 17; i++ {
		port := i + 9000
		n := getNode(strconv.Itoa(port))
		b.addNode(n)

		if i == 0 {
			n1 = n
		}
	}

	assert.Equal(t, b.size(), bucketSize)
	assert.Equal(t, b.findNode(n1), -1)
}

func Test_Bucket_GetRandNodes(t *testing.T) {
	b := getBuckets()

	n := getNode("9000")
	b.addNode(n)
	n = getNode("9001")
	b.addNode(n)

	nodes := b.getRandNodes(0)
	assert.Equal(t, len(nodes), 0)

	nodes = b.getRandNodes(1)
	assert.Equal(t, len(nodes), 1)

	nodes = b.getRandNodes(2)
	assert.Equal(t, len(nodes), 2)
	assert.Equal(t, isUniqueNodes(nodes), true)

	nodes = b.getRandNodes(3)
	assert.Equal(t, len(nodes), 2)
	assert.Equal(t, isUniqueNodes(nodes), true)
}

func Test_Bucket_GetRandNumbers(t *testing.T) {
	// Case 1: uppderBound < len
	rands := getRandNumbers(1, 2)
	assert.Equal(t, len(rands), 0)

	// Case 2: len == 0
	rands = getRandNumbers(1, 0)
	assert.Equal(t, len(rands), 0)

	// Case 2: len < 0
	rands = getRandNumbers(1, -1)
	assert.Equal(t, len(rands), 0)

	// valid inputs
	rands = getRandNumbers(10, 1)
	assert.Equal(t, len(rands), 1)

	rands = getRandNumbers(10, 2)
	assert.Equal(t, len(rands), 2)
	assert.Equal(t, isUniqueNumbers(rands), true)

	rands = getRandNumbers(10, 10)
	assert.Equal(t, len(rands), 10)
	assert.Equal(t, isUniqueNumbers(rands), true)
}

func Test_Bucket_Get(t *testing.T) {
	b := getBuckets()
	assert.Equal(t, b.get(0) == nil, true)
	assert.Equal(t, b.get(1) == nil, true)

	n1 := getNode("9000")
	b.addNode(n1)
	n2 := getNode("9001")
	b.addNode(n2)

	assert.Equal(t, b.get(0), n1)
	assert.Equal(t, b.get(1), n2)
}

func getNode(port string) *Node {
	id, err := crypto.GenerateRandomAddress()
	if err != nil {
		panic(err)
	}

	addr, _ := net.ResolveUDPAddr("udp", ":"+port)
	n := NewNode(*id, addr.IP, addr.Port, 0)

	return n
}

func isUniqueNumbers(rands []int) bool {
	generated := make(map[int]bool)

	for i := 0; i < len(rands); i++ {
		if generated[rands[i]] {
			return false
		}
		generated[rands[i]] = true
	}

	return true
}

func isUniqueNodes(nodes []*Node) bool {
	generated := make(map[common.Address]bool)

	for i := 0; i < len(nodes); i++ {
		if generated[nodes[i].ID] {
			return false
		}
		generated[nodes[i].ID] = true
	}

	return true
}
