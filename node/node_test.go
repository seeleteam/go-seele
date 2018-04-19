/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"testing"

	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
)

var (
	testNodeKey, _ = crypto.GenerateKey()
)

func testNodeConfig() *Config {
	return &Config{
		Name:    "test node",
		Version: "test version",
		P2P:     p2p.Config{PrivateKey: testNodeKey},
	}
}

// TestServiceA is a test implementation of the Service interface.
type TestServiceA struct{}

func (s TestServiceA) Protocols() []p2p.Protocol { return nil }
func (s TestServiceA) APIs() []rpc.API           { return nil }
func (s TestServiceA) Start(*p2p.Server) error   { return nil }
func (s TestServiceA) Stop() error               { return nil }

// TestServiceB is a test implementation of the Service interface.
type TestServiceB struct{}

func (s TestServiceB) Protocols() []p2p.Protocol { return nil }
func (s TestServiceB) APIs() []rpc.API           { return nil }
func (s TestServiceB) Start(*p2p.Server) error   { return nil }
func (s TestServiceB) Stop() error               { return nil }

// TestServiceC is a test implementation of the Service interface.
type TestServiceC struct{}

func (s TestServiceC) Protocols() []p2p.Protocol { return nil }
func (s TestServiceC) APIs() []rpc.API           { return nil }
func (s TestServiceC) Start(*p2p.Server) error   { return nil }
func (s TestServiceC) Stop() error               { return nil }

var testServiceA TestServiceA
var testServiceB TestServiceB
var testServiceC TestServiceC

func Test_ServiceRegistry(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}
	// Register a batch of services
	services := []Service{testServiceA, testServiceB, testServiceC}
	for i, service := range services {
		if err := stack.Register(service); err != nil {
			t.Fatalf("service #%d: registration failed: %v", i, err)
		}
	}
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start service stack: %v", err)
	}
	if err := stack.Stop(); err != nil {
		t.Fatalf("failed to stop service stack: %v", err)
	}
}
