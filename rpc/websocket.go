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
	"net/rpc/jsonrpc"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WSServer represents a Websocket RPC server
type WSServer struct {
	rpc.Server
}

// WebsocketServerConn represents a websocket server connection
type WebsocketServerConn struct {
	Ws *websocket.Conn
	r  io.Reader
	w  io.WriteCloser
}

// NewWSServer return a Websocket RPC server
func NewWSServer() *WSServer {
	server := &WSServer{
		rpc.Server{},
	}

	return server
}

// ServeWS runs the JSON-RPC server on a single websocket connection.
func ServeWS(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	defer ws.Close()

	if err != nil {
		log.Println(err)
		return
	}

	jsonrpc.ServeConn(ws.UnderlyingConn())
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
