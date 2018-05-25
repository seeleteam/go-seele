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
	"net/http"
	netrpc "net/rpc"
	"reflect"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
)

// error infos
var (
	ErrConfigIsNull       = errors.New("config info is null")
	ErrLogIsNull          = errors.New("SeeleLog is null")
	ErrNodeRunning        = errors.New("node is already running")
	ErrNodeStopped        = errors.New("node is not started")
	ErrServiceStartFailed = errors.New("node service starting failed")
	ErrServiceStopFailed  = errors.New("node service stopping failed")
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

	rpcAPIs []rpc.API

	log  *log.SeeleLog
	lock sync.RWMutex
}

// New creates a new P2P node.
func New(conf *Config) (*Node, error) {
	confCopy := *conf
	conf = &confCopy
	nlog := log.GetLogger("node", conf.LogConfig.PrintLog)

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
		specificShard = uint(rand.Intn(common.ShardNumber) + 1)
	}

	common.LocalShardNumber = specificShard

	if !n.config.SeeleConfig.Coinbase.Equal(common.Address{}) {
		coinbaseShard := common.GetShardNumber(n.config.SeeleConfig.Coinbase)
		if specificShard != 0 && coinbaseShard != specificShard {
			return errors.New(fmt.Sprintf("coinbase is not matched with specific shard number, "+
				"coinbase shard:%d, specific shard number:%d", coinbaseShard, specificShard))
		}
	}

	protocols := make([]p2p.Protocol, 0)
	for _, service := range n.services {
		protocols = append(protocols, service.Protocols()...)
	}

	p2pSever := p2p.NewServer(n.config.P2PConfig, protocols)
	if err := p2pSever.Start(n.config.SeeleConfig.GenesisConfig.ShardNumber); err != nil {
		return ErrServiceStartFailed
	}

	// Start services
	for i, service := range n.services {
		if err := service.Start(p2pSever); err != nil {
			for j := 0; j < i; j++ {
				n.services[j].Stop()
			}

			// stop the p2p server
			p2pSever.Stop()

			return err
		}
	}

	// Start RPC server
	if err := n.startRPC(n.services, n.config); err != nil {
		for _, service := range n.services {
			service.Stop()
		}

		// stop the p2p server
		p2pSever.Stop()

		return err
	}

	n.server = p2pSever

	return nil
}

// startRPC starts all RPCs
func (n *Node) startRPC(services []Service, conf *Config) error {
	apis := []rpc.API{}
	for _, service := range services {
		apis = append(apis, service.APIs()...)
	}

	if err := n.startJSONRPC(apis); err != nil {
		n.log.Error("starting json rpc failed", err)
		return err
	}

	if err := n.startHTTPRPC(apis, conf.HTTPServer.HTTPWhiteHost, conf.HTTPServer.HTTPCors); err != nil {
		n.log.Error("starting http rpc failed", err)
		return err
	}

	if err := n.startWSRPC(apis); err != nil {
		n.log.Error("start websocket err", err)
		return err
	}

	return nil
}

// startJSONRPC starts the json rpc server
func (n *Node) startJSONRPC(apis []rpc.API) error {
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			n.log.Error("Api registration failed", "service", api.Service, "namespace", api.Namespace)
			return err
		}
		n.log.Debug("registered service namespace: %s in json rpc successful", api.Namespace)
	}

	var (
		listerner net.Listener
		err       error
	)

	if listerner, err = net.Listen("tcp", n.config.BasicConfig.RPCAddr); err != nil {
		n.log.Error("Listening failed", "err", err)
		return err
	}

	n.log.Debug("Listerner address %s", listerner.Addr().String())
	go func() {
		for {
			conn, err := listerner.Accept()
			if err != nil {
				n.log.Error("RPC accepting failed", "err", err)
				continue
			}
			go handler.ServeCodec(rpc.NewJSONCodec(conn, nil))
		}
	}()

	return nil
}

// startHTTPRPC starts the http rpc server
func (n *Node) startHTTPRPC(apis []rpc.API, whitehosts []string, corsList []string) error {
	httpServer, httpHandler := rpc.NewHTTPServer(whitehosts, corsList)
	rpcServer := httpServer.GetRPCServer()
	for _, api := range apis {
		if err := rpcServer.RegisterName(api.Namespace, api.Service); err != nil {
			n.log.Error("Api registration failed", "service", api.Service, "namespace", api.Namespace)
			return err
		}
		n.log.Debug("registered service namespace: %s in http rpc successful", api.Namespace)
	}

	var (
		listerner net.Listener
		err       error
	)
	rpcServer.HandleHTTP(netrpc.DefaultRPCPath, netrpc.DefaultDebugPath)
	if listerner, err = net.Listen("tcp", n.config.HTTPServer.HTTPAddr); err != nil {
		n.log.Error("HTTP listening failed", "err", err)
		return err
	}

	go http.Serve(listerner, httpHandler)

	return nil
}

// startWSRPC starts websocket rpc server
func (n *Node) startWSRPC(apis []rpc.API) error {
	handler := rpc.NewWsRPCServer()
	rpcServer := handler.GetWsRPCServer()
	for _, api := range apis {
		if err := rpcServer.RegisterName(api.Namespace, api.Service); err != nil {
			n.log.Error("Websocket registration failed", "service", api.Service, "namespace", api.Namespace)
			return err
		}
	}
	http.HandleFunc(n.config.WSServerConfig.WSPattern, handler.ServeWS)
	go http.ListenAndServe(n.config.WSServerConfig.WSAddr, nil)

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

// Restart stops the running node and starts it again.
func (n *Node) Restart() error {
	if err := n.Stop(); err != nil {
		return err
	}
	if err := n.Start(); err != nil {
		return err
	}
	return nil
}
