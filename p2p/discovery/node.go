/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"fmt"
	"net"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

type Node struct {
	ID      NodeID //public key actually
	IP      net.IP
	UDPPort uint16

	sha common.Hash // node id for Kademila, which is generated from public key
}

const nodeIDBits = 512 // the length of the public key

type NodeID [nodeIDBits / 8]byte // we use public key as node id

// BytesTOID converts a byte slice to a NodeID
func BytesTOID(b []byte) (NodeID, error) {
	var id NodeID
	if len(b) != len(id) {
		return id, fmt.Errorf("wrong length, want %d bytes", len(id))
	}
	copy(id[:], b)
	return id, nil
}

func (id *NodeID) Bytes() []byte {
	return id[:]
}

func NewNode(id NodeID, addr *net.UDPAddr) *Node {
	return &Node{
		ID:      id,
		IP:      addr.IP,
		UDPPort: uint16(addr.Port),

		sha: crypto.Keccak256Hash(id[:]),
	}
}
