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
	"github.com/seeleteam/go-seele/log"
)

type Node struct {
	ID      NodeID //public key actually
	IP      net.IP
	UDPPort uint16

	// node id for Kademila, which is generated from public key
	// better to get it with getSha()
	sha *common.Hash
}

const nodeIDBits = 512 // the length of the public key

type NodeID [nodeIDBits / 8]byte // we use public key as node id

// BytesToID converts a byte slice to a NodeID
func BytesToID(b []byte) (NodeID, error) {
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

func (id *NodeID) ToSha() *common.Hash {
	hash := crypto.Keccak256Hash(id[:])
	return &hash
}

func (n *Node) getUDPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:n.IP,
		Port: int(n.UDPPort),
	}
}

func (n *Node) getSha() *common.Hash {
	if n.sha == nil || len(n.sha) == 0 {
		n.sha = n.ID.ToSha()
	}

	return n.sha
}

func NewNodeWithAddr(id NodeID, addr *net.UDPAddr) *Node {
	return NewNode(id, addr.IP, uint16(addr.Port))
}

func NewNode(id NodeID, ip net.IP, port uint16) *Node {
	return &Node{
		ID:      id,
		IP:      ip,
		UDPPort: port,
	}
}

func getRandomNodeID() NodeID {
	keypair, err := crypto.GenerateKey()
	if err != nil {
		log.Info(err.Error())
	}

	buff := crypto.FromECDSAPub(&keypair.PublicKey)

	id, err := BytesToID(buff[1:])
	if err != nil {
		log.Fatal(err.Error())
	}

	return id
}

