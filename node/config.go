/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/seele"
)

// Config holds the options for Node
type Config struct {
	// The name of the node
	Name string

	// The version of the node
	Version string

	// The file system path of the node, used to store data
	DataDir string

	// The configuration of p2p network
	P2P p2p.Config

	// RPCAddr is the address on which to start RPC server.
	RPCAddr string

	// HTTPAddr is the address of HTTP rpc server.
	HTTPAddr string

	// HTTPCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please note that CORS is a browser-enforced security, it's fully
	// useless for custom HTTP clients.
	HTTPCors []string

	// HTTPHostFilter is the whitelist of hostnames from which incoming requests are allowed.
	HTTPWhiteHost []string

	// The SeeleConfig is the configuration to create the seele service.
	SeeleConfig seele.Config
}
