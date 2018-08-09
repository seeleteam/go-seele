/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	rpc "github.com/seeleteam/go-seele/rpc2"
)

// error infos
var (
	ErrConfigIsNull       = errors.New("config info is null")
	ErrLogIsNull          = errors.New("SeeleLog is null")
	ErrNodeRunning        = errors.New("node is already running")
	ErrNodeStopped        = errors.New("node is not started")
	ErrServiceStartFailed = errors.New("failed to start node service")
	ErrServiceStopFailed  = errors.New("failed to stop node service")
)

// StopError represents an error which is returned when a node fails to stop any registered service
type StopError struct {
	Services map[reflect.Type]error // Services is a container mapping the type of services which fail to stop to error
}

// Error returns a string representation of the stop error.
func (se *StopError) Error() string {
	return fmt.Sprintf("services: %v", se.Services)
}

// Node is a container for registering services.
type Node struct {
	config *Config

	server   *p2p.Server
	services []Service

	log  *log.SeeleLog
	lock sync.RWMutex

	rpcListener net.Listener // IPC RPC listener socket to serve API requests
	rpcHandler  *rpc.Server  // IPC RPC request handler to process the API requests

	httpEndpoint string       // HTTP endpoint (interface + port) to listen at (empty = HTTP disabled)
	httpListener net.Listener // HTTP RPC listener socket to server API requests
	httpHandler  *rpc.Server  // HTTP RPC request handler to process the API requests

	wsEndpoint string       // Websocket endpoint (interface + port) to listen at (empty = websocket disabled)
	wsListener net.Listener // Websocket RPC listener socket to server API requests
	wsHandler  *rpc.Server  // Websocket RPC request handler to process the API requests
}

// New creates a new P2P node.
func New(conf *Config) (*Node, error) {
	confCopy := *conf
	conf = &confCopy
	nlog := log.GetLogger("node")

	return &Node{
		config:   conf,
		services: []Service{},
		log:      nlog,
	}, nil
}

// Register appends a new service into the node's stack.
func (n *Node) Register(service Service) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.server != nil {
		return ErrNodeRunning
	}
	n.services = append(n.services, service)

	return nil
}

// Start starts the p2p node.
func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.server != nil {
		return ErrNodeRunning
	}

	//check config
	specificShard := n.config.SeeleConfig.GenesisConfig.ShardNumber
	if specificShard == 0 {
		// select a shard randomly
		specificShard = uint(rand.Intn(common.ShardCount) + 1)
	}

	if specificShard > common.ShardCount {
		return fmt.Errorf("unsupported shard number, it must be in range [0, %d]", common.ShardCount)
	}

	common.LocalShardNumber = specificShard
	n.log.Info("local shard number is %d", common.LocalShardNumber)

	if !n.config.SeeleConfig.Coinbase.Equal(common.Address{}) {
		coinbaseShard := n.config.SeeleConfig.Coinbase.Shard()
		n.log.Info("coinbase is %s", n.config.SeeleConfig.Coinbase.ToHex())

		if coinbaseShard != specificShard {
			return fmt.Errorf("coinbase does not match with specific shard number, "+
				"coinbase shard:%d, specific shard number:%d", coinbaseShard, specificShard)
		}
	}

	protocols := make([]p2p.Protocol, 0)
	for _, service := range n.services {
		protocols = append(protocols, service.Protocols()...)
	}

	p2pServer := p2p.NewServer(n.config.SeeleConfig.GenesisConfig, n.config.P2PConfig, protocols)
	if err := p2pServer.Start(n.config.BasicConfig.DataDir, n.config.SeeleConfig.GenesisConfig.ShardNumber); err != nil {
		return ErrServiceStartFailed
	}

	// Start services
	for i, service := range n.services {
		if err := service.Start(p2pServer); err != nil {
			n.log.Error("got error when start service %s", err)

			for j := 0; j < i; j++ {
				n.services[j].Stop()
			}

			// stop the p2p server
			p2pServer.Stop()

			return err
		}
	}

	// Start RPC server
	if err := n.startRPC(n.services); err != nil {
		n.log.Error("got error when start rpc %s", err)

		for _, service := range n.services {
			service.Stop()
		}

		// stop the p2p server
		p2pServer.Stop()

		return err
	}

	n.server = p2pServer

	return nil
}

// Stop terminates the running node and services registered.
func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.server == nil {
		return ErrNodeStopped
	}

	// stopErr is intended for possible stop errors
	stopErr := &StopError{
		Services: make(map[reflect.Type]error),
	}

	for _, service := range n.services {
		if err := service.Stop(); err != nil {
			stopErr.Services[reflect.TypeOf(service)] = err
		}
	}

	// stop the p2p server
	n.server.Stop()

	n.services = nil
	n.server = nil

	// return the stop errors if any
	if len(stopErr.Services) > 0 {
		return stopErr
	}

	return nil
}
