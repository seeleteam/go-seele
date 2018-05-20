/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

var (
	invalidNodeError = "invalid node"
	nodeHeaderError  = "node id should start with snode://"

	nodeHeader = "snode://"
)

// Node the node that contains its public key and network address
type Node struct {
	ID               common.Address //public key actually
	IP               net.IP
	UDPPort, TCPPort int

	Shard		int //node shard number

	// node id for Kademila, which is generated from public key
	// better to get it with getSha()
	sha common.Hash
}

// NewNode new node with its value
func NewNode(id common.Address, ip net.IP, port int, shard int) *Node {
	return &Node{
		ID:      id,
		IP:      ip,
		UDPPort: port,
		Shard:shard,
	}
}

// NewNodeWithAddr new node with id and network address
func NewNodeWithAddr(id common.Address, addr *net.UDPAddr, shard int) *Node {
	return NewNode(id, addr.IP, addr.Port, shard)
}

func NewNodeFromString(id string) (*Node, error) {
	if !strings.HasPrefix(id, nodeHeader) {
		return nil, errors.New(nodeHeaderError)
	}

	// cut prefix header
	id = id[len(nodeHeader):]

	idSplit := strings.Split(id, "@")
	if len(idSplit) != 2 {
		return nil, errors.New(invalidNodeError)
	}

	address, err := hex.DecodeString(idSplit[0])
	if err != nil {
		return nil, err
	}

	publicKey, err := common.NewAddress(address)
	if err != nil {
		return nil, err
	}

	addr, err := net.ResolveUDPAddr("udp", idSplit[1])
	if err != nil {
		return nil, err
	}

	node := NewNodeWithAddr(publicKey, addr, 0)
	return node, nil
}

func (n *Node) GetUDPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   n.IP,
		Port: int(n.UDPPort),
	}
}

func (n *Node) setShard(shard int) {
	n.Shard = shard
}

func (n *Node) getSha() common.Hash {
	if n.sha == common.EmptyHash {
		n.sha = crypto.HashBytes(n.ID.Bytes())
	}

	return n.sha
}

func (n *Node) String() string {
	addr := n.GetUDPAddr().String()
	hex := hex.EncodeToString(n.ID.Bytes())
	return fmt.Sprintf(nodeHeader+"%s@%s", hex, addr)
}
