/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"errors"
	"sync"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
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

	log  *log.SeeleLog
	lock sync.RWMutex
}

// New creates a new P2P node.
func New(conf *Config) (*Node, error) {
	confCopy := *conf
	conf = &confCopy

	return &Node{
		config:   conf,
		services: []Service{},
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

	//Start services
	for i, service := range n.services {
		if err := service.Start(running); err != nil {
			for j := 0; j < i; j++ {
				service.Stop()
			}
			//running.Stop() need add later
			return err
		}
	}
	n.server = running

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
	//n.server.stop() need add later
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
