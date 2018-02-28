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

var (
	ErrNodeRunning = errors.New("node is already running")
	ErrNodeStopped = errors.New("node is not started")
)

// Node is a container for registering services.
type Node struct {
	config       *Config
	serverConfig p2p.Config
	server       *p2p.Server

	log  *log.SeeleLog
	lock sync.RWMutex
}

// Start create a p2p node.
func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.server != nil {
		return ErrNodeRunning
	}

	return nil
}

// Stop terminates the running the node and the services registered.
func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.server == nil {
		return ErrNodeStopped
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
