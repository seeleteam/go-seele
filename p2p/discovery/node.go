/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

// Node the node that contains its public key and network address
type Node struct {
	ID      common.Address //public key actually
	IP      net.IP
	UDPPort uint16

	// node id for Kademila, which is generated from public key
	// better to get it with getSha()
	sha *common.Hash
}

func (n *Node) getUDPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   n.IP,
		Port: int(n.UDPPort),
	}
}

func (n *Node) getSha() *common.Hash {
	if n.sha == nil || len(n.sha) == 0 {
		n.sha = n.ID.ToSha()
	}

	return n.sha
}

// NewNodeWithAddr new node with id and network address
func NewNodeWithAddr(id common.Address, addr *net.UDPAddr) *Node {
	return NewNode(id, addr.IP, uint16(addr.Port))
}

// NewNode new node with its value
func NewNode(id common.Address, ip net.IP, port uint16) *Node {
	return &Node{
		ID:      id,
		IP:      ip,
		UDPPort: port,
	}
}

