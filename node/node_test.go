/**
*  @file
*  @copyright defined in go-seele/LICENSE
*/

package node

import (
	"testing"

	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/p2p"
)

var(
	testNodeKey, _ = crypto.GenerateKey()
)
	
func testNodeConfig() *Config {
	return &Config{
		Name: "test node",
		P2P: p2p.Config{PrivateKey: testNodeKey},
	}
}

func Test_ServiceRegistry(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Register a batch of unique services and ensure they start successfully
	services := []ServiceConstructor{NewNoopServiceA, NewNoopServiceB, NewNoopServiceC}
	for i, constructor := range services {
		if err := stack.Register(constructor); err != nil {
			t.Fatalf("service #%d: registration failed: %v", i, err)
		}
	}
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start original service stack: %v", err)
	}
	if err := stack.Register(NewNoopServiceB); err != nil {
		t.Fatalf("duplicate registration failed: %v", err)
	}
}
