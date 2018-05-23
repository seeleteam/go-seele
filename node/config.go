/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
)

// Config is the Configuration of node
type Config struct {
	//Config is the Configuration of log
	LogConfig comm.LogConfig

	// basic config for Node
	BasicConfig BasicConfig

	// The configuration of p2p network
	P2PConfig p2p.Config

	// HttpServer config for http server
	HTTPServer HTTPServer

	// The SeeleConfig is the configuration to create the seele service.
	SeeleConfig SeeleConfig

	// The configuration of websocket rpc service
	WSServerConfig rpc.WSServerConfig
}

// BasicConfig config for Node
type BasicConfig struct {
	// The name of the node
	Name string `json:"name"`

	// The version of the node
	Version string `json:"version"`

	// The file system path of the node, used to store data
	DataDir string `json:"dataDir"`

	// RPCAddr is the address on which to start RPC server.
	RPCAddr string `json:"address"`

	// coinbase used by the miner
	Coinbase string `json:"coinbase"`
}

// HTTPServer config for http server
type HTTPServer struct {
	// The HTTPAddr is the address of HTTP rpc service
	HTTPAddr string `json:"address"`

	// HTTPCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	HTTPCors []string `json:"crosssorgins"`

	// HTTPHostFilter is the whitelist of hostnames which are allowed on incoming requests.
	HTTPWhiteHost []string `json:"whiteHost"`
}

// Config is the seele's configuration to create seele service
type SeeleConfig struct {
	TxConf core.TransactionPoolConfig

	Coinbase common.Address

	GenesisConfig core.GenesisInfo
}
