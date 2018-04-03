/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"errors"
	"net"
	"sync"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
)

// error infos
var (
	ErrNodeRunning        = errors.New("node is already running")
	ErrNodeStopped        = errors.New("node is not started")
	ErrServiceStartFailed = errors.New("node service start failed")
	ErrServiceStopFailed  = errors.New("node service stop failed")
)

// Node is a container for registering services.
type Node struct {
	config *Config

	serverConfig p2p.Config
	server       *p2p.Server

	services []Service

	rpcAPIs []rpc.API

	log  *log.SeeleLog
	lock sync.RWMutex
}

// New creates a new P2P node.
func New(conf *Config) (*Node, error) {
	confCopy := *conf
	conf = &confCopy
	nlog := log.GetLogger("node", true)

	return &Node{
		config:   conf,
		services: []Service{},
		log:      nlog,
	}, nil
}

// Register append a new service into the node's stack.
func (n *Node) Register(service Service) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.server != nil {
		return ErrNodeRunning
	}
	n.services = append(n.services, service)

	return nil
}

// Start create a p2p node.
func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.server != nil {
		return ErrNodeRunning
	}

	n.serverConfig = n.config.P2P
	running := &p2p.Server{Config: n.serverConfig}
	for _, service := range n.services {
		running.Protocols = append(running.Protocols, service.Protocols()...)
	}
	if err := running.Start(); err != nil {
		return ErrServiceStartFailed
	}

	// Start services
	for i, service := range n.services {
		if err := service.Start(running); err != nil {
			for j := 0; j < i; j++ {
				service.Stop()
			}

			return err
		}
	}

	// Start RPC server
	if err := n.startRPC(n.services); err != nil {
		for _, service := range n.services {
			service.Stop()
		}
		return err
	}

	n.server = running

	return nil
}

// startRPC starts all RPC
func (n *Node) startRPC(services []Service) error {
	apis := []rpc.API{}
	for _, service := range services {
		apis = append(apis, service.APIs()...)
	}

	if err := n.startJSONRPC(apis); err != nil {
		n.log.Error("startProc err", err)
		return err
	}

	return nil
}

// startJSONRPC starts JSONRPC server
func (n *Node) startJSONRPC(apis []rpc.API) error {
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			n.log.Error("Api registered failed", "service", api.Service, "namespace", api.Namespace)
			return err
		}
		n.log.Debug("Proc registered service namespace %s", api.Namespace)
	}

	var (
		listerner net.Listener
		err       error
	)

	if listerner, err = net.Listen("tcp", n.config.RPCAddr); err != nil {
		n.log.Error("Listen failed", "err", err)
		return err
	}

	n.log.Debug("Listerner address %s", listerner.Addr().String())
	go func() {
		for {
			conn, err := listerner.Accept()
			if err != nil {
				n.log.Error("RPC accept failed", "err", err)
				continue
			}
			go handler.ServeCodec(rpc.NewJsonCodec(conn))
		}
	}()

	return nil
}

// Stop terminates the running the node and the services registered.
func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.server == nil {
		return ErrNodeStopped
	}
	for _, service := range n.services {
		if err := service.Stop(); err != nil {
			return ErrNodeStopped
		}
	}

	n.services = nil
	n.server = nil

	return nil
}

// Restart stop a running node and start a new one.
func (n *Node) Restart() error {
	if err := n.Stop(); err != nil {
		return err
	}
	if err := n.Start(); err != nil {
		return err
	}
	return nil
}
