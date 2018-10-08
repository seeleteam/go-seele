/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"strings"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/stretchr/testify/assert"
)

var (
	testNodeKey, _ = crypto.GenerateKey()
)

func testNodeConfig() *Config {
	return &Config{
		BasicConfig: BasicConfig{
			Name:    "test node",
			Version: "test version",
		},
		P2PConfig: p2p.Config{PrivateKey: testNodeKey},
		WSServerConfig: WSServerConfig{
			Address:      "127.0.0.1:8080",
			CrossOrigins: []string{"*"},
		},
		LogConfig: comm.LogConfig{PrintLog: true, IsDebug: true},
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

	err = stack.Register(services[0])
	if err == nil || err != ErrNodeRunning {
		t.Fatalf("expected ErrNodeRunning error when node is already running")
	}

	if err := stack.Stop(); err != nil {
		t.Fatalf("failed to stop service stack: %v", err)
	}
}

func Test_ServiceStart(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start service stack: %v", err)
	}

	err = stack.Start()
	if err == nil || err != ErrNodeRunning {
		t.Fatalf("expected ErrNodeRunning error when node is already running")
	}

	if err := stack.Stop(); err != nil {
		t.Fatalf("failed to stop service stack: %v", err)
	}

	// unsupported shard number
	stack.config.SeeleConfig.GenesisConfig.ShardNumber = 21
	err = stack.checkConfig()
	assert.Equal(t, err != nil, true)
	assert.Equal(t, strings.Contains(err.Error(), "unsupported shard number"), true)

	// coinbase does not match with specific shard number
	stack.config.SeeleConfig.GenesisConfig.ShardNumber = 2
	stack.config.SeeleConfig.Coinbase = common.BytesToAddress([]byte("testAddr"))
	err = stack.checkConfig()
	assert.Equal(t, err != nil, true)
	assert.Equal(t, strings.Contains(err.Error(), "coinbase does not match with specific shard number"), true)

	// started normally
	stack.config.SeeleConfig.GenesisConfig.ShardNumber = 1
	stack.config.SeeleConfig.Coinbase = common.BytesToAddress([]byte("testAddr"))

	// Register a batch of services
	services := []Service{testServiceA, testServiceB, testServiceC}
	for i, service := range services {
		if err := stack.Register(service); err != nil {
			t.Fatalf("service #%d: registration failed: %v", i, err)
		}
	}
	err = stack.Start()
	assert.Equal(t, err, nil)

	if err := stack.Stop(); err != nil {
		t.Fatalf("failed to stop service stack: %v", err)
	}
}
