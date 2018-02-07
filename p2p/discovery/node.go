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

func (id *NodeID) ToSha() common.Hash  {
	return crypto.Keccak256Hash(id[:])
}

func NewNodeWithAddr(id NodeID, addr *net.UDPAddr) *Node {
	return NewNode(id, addr.IP, uint16(addr.Port))
}

func NewNode(id NodeID, ip net.IP, port uint16) *Node {
	return &Node{
		ID:      id,
		IP:      ip,
		UDPPort: port,

		sha: id.ToSha(),
	}
}
