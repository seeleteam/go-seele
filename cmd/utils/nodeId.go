/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package utils

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/seeleteam/go-seele/common"
	"net"
	"strings"
)

var (
	invalidNodeError string = "invalid node"
	nodeHeaderError  string = "node id should start with snode://"

	nodeHeader string = "snode://"
)

// node id for command line, it contains public key, ip and port
type nodeId struct {
	Address common.Address
	IP      net.IP	// only support ipv4 for now
	Port    int
}

func NewNodeId(id string) (*nodeId, error) {
	if !strings.HasPrefix(id, nodeHeader) {
		return nil,errors.New(nodeHeaderError)
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
		return  nil, err
	}

	addr, err := net.ResolveUDPAddr("udp", idSplit[1])
	if err != nil {
		return  nil, err
	}

	node := &nodeId{
		Address: publicKey,
		IP:      addr.IP,
		Port:    addr.Port,
	}

	return node, nil
}

func (n *nodeId) GetUDPAddr() *net.UDPAddr  {
	return &net.UDPAddr{
		IP: n.IP,
		Port: n.Port,
	}
}

func (n *nodeId) String() string {
	return fmt.Sprintf(nodeHeader + "%s@%s", hex.EncodeToString(n.Address.Bytes()), n.GetUDPAddr().String())
}