/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"github.com/seeleteam/go-seele/common"
	"net"
	"crypto"
)

type Node struct {
	ID NodeID //public key actually
	IP net.IP
	UDPPort uint16

	sha common.Hash // node id for Kademila, which is generate from public key
}

const nodeIDBits = 512 // the length of the public key

type NodeID [nodeIDBits/8]byte // we use public key as node id

func NewNode(id NodeID, addr *net.UDPAddr) *Node  {
	return &Node {
		ID: id,
		IP: addr.IP,
		UDPPort: addr.Port,

		sha: crypto.Keccak256Hash(id[:]),
	}
}