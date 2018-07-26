/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"net"
	"strings"

	rpc "github.com/seeleteam/go-seele/rpc2"
)

// startRPC is a helper method to start all the various RPC endpoint during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (n *Node) startRPC(services []Service) error {
	// Gather all the possible APIs to surface
	apis := []rpc.API{}
	for _, service := range services {
		apis = append(apis, service.APIs()...)
	}

	// Start the various API endpoints, terminating all in case of errors
	if err := n.startPRC(apis); err != nil {
		return err
	}

	if err := n.startHTTP(apis); err != nil {
		n.stopRPC()
		return err
	}

	if err := n.startWS(apis); err != nil {
		n.stopHTTP()
		n.stopRPC()
		return err
	}

	return nil
}

// startIPC initializes and starts the IPC RPC endpoint.
func (n *Node) startPRC(apis []rpc.API) error {
	endpoint := n.config.BasicConfig.RPCAddr
	// Short circuit if the IPC endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}

	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		n.log.Debug("registered RPC service namespace %s", api.Namespace)
	}

	// All APIs registered, start the IPC listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}

	go func() {
		n.log.Info("RPC opened at address %s", endpoint)

		for {
			conn, err := listener.Accept()
			if err != nil {
				// Terminate if the listener was closed
				n.lock.RLock()
				closed := n.rpcListener == nil
				n.lock.RUnlock()
				if closed {
					return
				}
				// Not closed, just some error; report and continue
				n.log.Error("failed to accept RPC. err %s", err)
				continue
			}
			go handler.ServeCodec(rpc.NewJSONCodec(conn), rpc.OptionMethodInvocation|rpc.OptionSubscriptions)
		}
	}()

	// All listeners booted successfully
	n.rpcListener = listener
	n.rpcHandler = handler

	return nil
}

// stopRPC terminates the IPC RPC endpoint.
func (n *Node) stopRPC() {
	if n.rpcListener != nil {
		n.rpcListener.Close()
		n.rpcListener = nil

		n.log.Info("RPC closed")
	}
	if n.rpcHandler != nil {
		n.rpcHandler.Stop()
		n.rpcHandler = nil
	}
}

// startHTTP initializes and starts the HTTP RPC endpoint.
func (n *Node) startHTTP(apis []rpc.API) error {
	endpoint := n.config.HTTPServer.HTTPAddr
	cors := n.config.HTTPServer.HTTPCors
	vhosts := n.config.HTTPServer.HTTPWhiteHost

	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}

	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if api.Public {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return err
			}
			n.log.Debug("HTTP registered service namespace %s", api.Namespace)
		}
	}

	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}

	go rpc.NewHTTPServer(cors, vhosts, handler).Serve(listener)
	n.log.Info("HTTP endpoint opened. url http://%s, cors %s, whitehost %s", endpoint, strings.Join(cors, ","), strings.Join(vhosts, ","))

	// All listeners booted successfully
	n.httpEndpoint = endpoint
	n.httpListener = listener
	n.httpHandler = handler

	return nil
}

// stopHTTP terminates the HTTP RPC endpoint.
func (n *Node) stopHTTP() {
	if n.httpListener != nil {
		n.httpListener.Close()
		n.httpListener = nil

		n.log.Info("HTTP endpoint closed. url http://%s", n.httpEndpoint)
	}
	if n.httpHandler != nil {
		n.httpHandler.Stop()
		n.httpHandler = nil
	}
}

// startWS initializes and starts the websocket RPC endpoint.
func (n *Node) startWS(apis []rpc.API) error {
	endpoint := n.config.WSServerConfig.Address
	wsOrigins := n.config.WSServerConfig.CrossOrigins

	// Short circuit if the WS endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}

	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if api.Public {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return err
			}
			n.log.Debug("WebSocket registered. service namespace %s", api.Namespace)
		}
	}

	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}

	go rpc.NewWSServer(wsOrigins, handler).Serve(listener)
	n.log.Info("WebSocket endpoint opened. url ws://%s", listener.Addr())

	// All listeners booted successfully
	n.wsEndpoint = endpoint
	n.wsListener = listener
	n.wsHandler = handler

	return nil
}

// stopWS terminates the websocket RPC endpoint.
func (n *Node) stopWS() {
	if n.wsListener != nil {
		n.wsListener.Close()
		n.wsListener = nil

		n.log.Info("WebSocket endpoint closed. url ws://%s", n.wsEndpoint)
	}
	if n.wsHandler != nil {
		n.wsHandler.Stop()
		n.wsHandler = nil
	}
}
