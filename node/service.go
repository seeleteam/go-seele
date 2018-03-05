/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"reflect"
	"github.com/seeleteam/go-seele/p2p"
)

// Service registers to node after node starts.
type Service interface {
	// Protocols retrieves the P2P protocols the service wishes to start.
	Protocols() []p2p.Protocol

	Start(server *p2p.Server) error

	Stop() error
}

type ServiceContext struct {
	config *Config
	services map[reflect.Type]Service
}

type ServiceConstructor func(ctx *ServiceContext) (Service, error)
