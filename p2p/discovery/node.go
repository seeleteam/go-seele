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
	"strconv"
	"strings"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

var (
	errInvalidNodeString = errors.New("invalid node string")
	errNodeHeaderInvalid = errors.New("node id should start with snode://")

	nodeHeader = "snode://"
)

// Node the node that contains its public key and network address
type Node struct {
	ID               common.Address //public key actually
	IP               net.IP
	UDPPort, TCPPort int

	Shard uint //node shard number

	// node id for Kademila, which is generated from public key
	// better to get it with getSha()
	sha common.Hash
}

// NewNode new node with its value
func NewNode(id common.Address, ip net.IP, port int, shard uint) *Node {
	return &Node{
		ID:      id,
		IP:      ip,
		UDPPort: port,
		Shard:   shard,
	}
}

// NewNodeWithAddr new node with id and network address
func NewNodeWithAddr(id common.Address, addr *net.UDPAddr, shard uint) *Node {
	return NewNode(id, addr.IP, addr.Port, shard)
}

func NewNodeFromIP(id string) (*Node, error) {
	addr, err := net.ResolveUDPAddr("udp", id)
	if err != nil {
		return nil, err
	}

	node := NewNodeWithAddr(common.Address{}, addr, 0)
	return node, nil
}

func NewNodeFromString(id string) (*Node, error) {
	if !strings.HasPrefix(id, nodeHeader) {
		return nil, errNodeHeaderInvalid
	}

	// cut prefix header
	id = id[len(nodeHeader):]

	// node id
	idSplit := strings.Split(id, "@")
	if len(idSplit) != 2 {
		return nil, errInvalidNodeString
	}

	nodeId, err := hex.DecodeString(idSplit[0])
	if err != nil {
		return nil, err
	}

	publicKey, err := common.NewAddress(nodeId)
	if err != nil {
		return nil, err
	}

	// udp address
	addrSplit := strings.Split(idSplit[1], "[")
	if len(addrSplit) != 2 {
		return nil, errInvalidNodeString
	}

	addr, err := net.ResolveUDPAddr("udp", addrSplit[0])
	if err != nil {
		return nil, err
	}

	// shard
	if len(addrSplit[1]) < 1 {
		return nil, errInvalidNodeString
	}

	shardStr := addrSplit[1][:len(addrSplit[1])-1]
	shard, err := strconv.Atoi(shardStr)
	if err != nil {
		return nil, err
	}

	node := NewNodeWithAddr(publicKey, addr, uint(shard))
	return node, nil
}

func (n *Node) GetUDPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   n.IP,
		Port: int(n.UDPPort),
	}
}

func (n *Node) setShard(shard uint) {
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
	return fmt.Sprintf(nodeHeader+"%s@%s[%d]", hex, addr, n.Shard)
}

func (node *Node) UnmarshalText(json []byte) error {
	n, err := NewNodeFromIP(string(json))
	if err != nil {
		return err
	}
	*node = *n
	return nil
}
