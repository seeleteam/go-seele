/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"errors"
	"testing"

	"github.com/seeleteam/go-seele/log/comm"
	"github.com/seeleteam/go-seele/p2p"
	rpc "github.com/seeleteam/go-seele/rpc"
	"github.com/stretchr/testify/assert"
)

// TestService1 is a test implementation of the Service interface.
type TestService1 struct{}

func (s TestService1) Protocols() []p2p.Protocol { return nil }
func (s TestService1) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "monitor",
			Version:   "1.0",
			Service:   testServiceA,
			Public:    true,
		},
	}
}

func (s TestService1) Start(*p2p.Server) error { return errors.New("failed to start server") }
func (s TestService1) Stop() error             { return nil }

var testService1 TestService1

func validTCPConfig() *Config {
	return &Config{
		BasicConfig: BasicConfig{
			Name:    "test node",
			Version: "test version",
			RPCAddr: "127.0.0.1:8080",
		},
		LogConfig: comm.LogConfig{PrintLog: true, IsDebug: true},
	}
}

func invalidTCPConfigWithoutEndpoint() *Config {
	return &Config{
		BasicConfig: BasicConfig{
			Name:    "test node",
			Version: "test version",
		},
		LogConfig: comm.LogConfig{PrintLog: true, IsDebug: true},
	}
}

func invalidTCPConfig() *Config {
	return &Config{
		BasicConfig: BasicConfig{
			Name:    "test node",
			Version: "test version",
			RPCAddr: "127.0.0.1",
		},
		LogConfig: comm.LogConfig{PrintLog: true, IsDebug: true},
	}
}

func validIPCConfig() *Config {
	return &Config{
		LogConfig: comm.LogConfig{PrintLog: true, IsDebug: true},
	}
}

func validHTTPConfig() *Config {
	return &Config{
		HTTPServer: HTTPServer{
			HTTPAddr: "127.0.0.1:8080",
			HTTPCors: []string{"*"},
		},
		LogConfig: comm.LogConfig{PrintLog: true, IsDebug: true},
	}
}

func invalidHTTPConfig() *Config {
	return &Config{
		HTTPServer: HTTPServer{
			HTTPAddr: "127.0.0.1",
		},
		LogConfig: comm.LogConfig{PrintLog: true, IsDebug: true},
	}
}

func validWSConfig() *Config {
	return &Config{
		WSServerConfig: WSServerConfig{
			Address:      "127.0.0.1:8080",
			CrossOrigins: []string{"*"},
		},
		LogConfig: comm.LogConfig{PrintLog: true, IsDebug: true},
	}
}

func invalidWSConfig() *Config {
	return &Config{
		WSServerConfig: WSServerConfig{
			Address: "127.0.0.1",
		},
		LogConfig: comm.LogConfig{PrintLog: true, IsDebug: true},
	}
}

func Test_startTCP(t *testing.T) {
	// valid node config
	stack := newNode(validTCPConfig(), t)
	apis := newAPIs()
	err := stack.startTCP(apis)
	assert.Equal(t, err, nil)
	stack.stopRPC()

	// invalid node config
	stack = newNode(invalidTCPConfigWithoutEndpoint(), t)
	err = stack.startTCP(apis)
	assert.Equal(t, err, nil)
	stack.stopRPC()

	// invalid node config
	stack = newNode(invalidTCPConfig(), t)
	err = stack.startTCP(apis)
	assert.Equal(t, err != nil, true)
	stack.stopRPC()

	// Stop node that not started
	err = stack.Stop()
	assert.Equal(t, err, ErrNodeStopped)
}

func Test_startIPC(t *testing.T) {
	// valid node config
	stack := newNode(validIPCConfig(), t)
	apis := newAPIs()
	err := stack.startIPC(apis)
	assert.Equal(t, err, nil)
	stack.stopRPC()

	// Stop node that not started
	err = stack.Stop()
	assert.Equal(t, err, ErrNodeStopped)
}

func Test_startHTTP(t *testing.T) {
	// valid node config
	stack := newNode(validHTTPConfig(), t)
	apis := newAPIs()
	err := stack.startHTTP(apis)
	assert.Equal(t, err, nil)
	stack.stopHTTP()

	// invalid node config
	stack = newNode(invalidHTTPConfig(), t)
	err = stack.startHTTP(apis)
	assert.Equal(t, err != nil, true)
	stack.stopHTTP()
}

func Test_startWS(t *testing.T) {
	// valid node config
	stack := newNode(validWSConfig(), t)
	apis := newAPIs()
	err := stack.startWS(apis)
	assert.Equal(t, err, nil)
	stack.stopWS()

	// invalid node config
	stack = newNode(invalidWSConfig(), t)
	err = stack.startWS(apis)
	assert.Equal(t, err != nil, true)
	stack.stopWS()
}

func newNode(config *Config, t *testing.T) *Node {
	stack, err := New(config)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	return stack
}

func newAPIs() []rpc.API {
	services := []Service{testService1}
	apis := []rpc.API{}

	for _, service := range services {
		apis = append(apis, service.APIs()...)
	}

	return apis
}
