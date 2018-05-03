/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"errors"
	"net"
	"net/http"
	netrpc "net/rpc"
	"reflect"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/seeleteam/go-seele/seele"
)

// error infos
var (
	ErrConfigIsNull       = errors.New("config info is null")
	ErrLogIsNull          = errors.New("SeeleLog is null")
	ErrNodeRunning        = errors.New("node is already running")
	ErrNodeStopped        = errors.New("node is not started")
	ErrServiceStartFailed = errors.New("node service start failed")
	ErrServiceStopFailed  = errors.New("node service stop failed")
)

// StopError represents an error which is returned when a node fails to stop any registered service
type StopError struct {
	Services map[reflect.Type]error
}

// Error returns a string representation of the stop error.
func (se *StopError) Error() string {
	return fmt.Sprintf("services: %v", se.Services)
}

// Node is a container for registering services.
type Node struct {
	config *Config

	serverConfig p2p.Config
	server       *p2p.Server

	services []Service

	rpcAPIs []rpc.API

	log  *log.SeeleLog
	lock sync.RWMutex

	miner *miner.Miner
}

// New creates a new P2P node.
func New(conf *Config) (*Node, error) {
	confCopy := *conf
	conf = &confCopy
	nlog := log.GetLogger("node", common.PrintLog)

	return &Node{
		config:   conf,
		services: []Service{},
		log:      nlog,
	}, nil
}

// Miner get miner info
func (n *Node) Miner() *miner.Miner { return n.miner }

// StartMiner create miner and begin to minning
func (n *Node) StartMiner(service *seele.SeeleService) error {
	if n.config == nil {
		return ErrConfigIsNull
	}

	if n.log == nil {
		return ErrLogIsNull
	}

	if n.miner == nil {
		n.miner = miner.NewMiner(n.config.SeeleConfig.Coinbase, service, n.log)
		go n.miner.Start()
	} else {
		if n.miner.IsMining() == false {
			go n.miner.Start()
		}
	}

	return nil
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
				n.services[j].Stop()
			}
            
			// stop the p2p server
			running.Stop()
			
			return err
		}
	}

	// Start RPC server
	if err := n.startRPC(n.services, n.config); err != nil {
		for _, service := range n.services {
			service.Stop()
		}
		
		// stop the p2p server
		running.Stop()
		
		return err
	}

	n.server = running

	return nil
}

// startRPC starts all RPC
func (n *Node) startRPC(services []Service, conf *Config) error {
	apis := []rpc.API{}
	for _, service := range services {
		apis = append(apis, service.APIs()...)
	}

	if err := n.startJSONRPC(apis); err != nil {
		n.log.Error("startProc err", err)
		return err
	}

	if err := n.startHTTPRPC(apis, conf.HTTPWhiteHost, conf.HTTPCors); err != nil {
		n.log.Error("start http rpc err", err)
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

// startHTTPRPC starts http rpc server
func (n *Node) startHTTPRPC(apis []rpc.API, whitehosts []string, corsList []string) error {
	httpServer, httpHandler := rpc.NewHTTPServer(whitehosts, corsList)
	for _, api := range apis {
		if err := httpServer.RegisterName(api.Namespace, api.Service); err != nil {
			n.log.Error("Api registered failed", "service", api.Service, "namespace", api.Namespace)
			return err
		}
		n.log.Debug("Proc registered service namespace %s", api.Namespace)
	}

	var (
		listerner net.Listener
		err       error
	)
	httpServer.HandleHTTP(netrpc.DefaultRPCPath, netrpc.DefaultDebugPath)
	if listerner, err = net.Listen("tcp", n.config.HTTPAddr); err != nil {
		n.log.Error("HTTP listen failed", "err", err)
		return err
	}

	go http.Serve(listerner, httpHandler)

	return nil
}

// Stop terminates the running the node and the services registered.
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
