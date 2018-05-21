/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package rpc

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/rpc"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WSServerConfig config for websocket server
type WSServerConfig struct {
	// The WSAddr is the address of Websocket rpc service
	WSAddr string `json:"address"`
	// The WSAddr is the pattern of Websocket rpc service
	WSPattern string `json:"pattern"`
}

// WsRPCServer represents a Websocket RPC server
type WsRPCServer struct {
	rpc *rpc.Server
}

// WebsocketServerConn represents a websocket server connection
type WebsocketServerConn struct {
	Ws *websocket.Conn
	r  io.Reader
	w  io.WriteCloser
}

// NewWsRPCServer return a Websocket RPC server
func NewWsRPCServer() *WsRPCServer {
	server := &WsRPCServer{
		rpc: &rpc.Server{},
	}

	return server
}

// GetWsRPCServer return rpc server of the WsRPCServer
func (server *WsRPCServer) GetWsRPCServer() *rpc.Server {
	return server.rpc
}

// ServeWS runs the JSON-RPC server on a single websocket connection.
func (server *WsRPCServer) ServeWS(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	defer ws.Close()

	if err != nil {
		log.Println(err)
		return
	}

	server.rpc.ServeCodec(NewJSONCodec(ws.UnderlyingConn(), nil))
}

// Read represents read data from websocket connection.
func (wc *WebsocketServerConn) Read(p []byte) (n int, err error) {
	if wc.r == nil {
		_, wc.r, err = wc.Ws.NextReader()

		if err != nil {
			fmt.Println(err)
			return 0, err
		}
	}

	n, err = wc.r.Read(p)
	if err == io.EOF {
		wc.r = nil
	}

	return
}

// Write represents write data for websocket connection.
func (wc *WebsocketServerConn) Write(p []byte) (n int, err error) {
	if wc.w == nil {
		wc.w, err = wc.Ws.NextWriter(websocket.TextMessage)
		if err != nil {
			return 0, err
		}
	}

	n, err = wc.w.Write(p)
	if err != nil || n == len(p) {
		err = wc.Close()
	}

	return

}

// Close represents close the websocket connection.
func (wc *WebsocketServerConn) Close() (err error) {
	if wc.w != nil {
		err = wc.w.Close()
		wc.w = nil
	}

	return
}
