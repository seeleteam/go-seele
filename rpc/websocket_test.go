/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package rpc

import (
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

type WSTest struct{}

func (t *WSTest) Echo(req *string, res *string) error {
	*res = *req
	return nil
}
func Test_Websocket(t *testing.T) {
	rpc.RegisterName("Test", new(WSTest))
	http.HandleFunc("/ws", ServeWS)
	go http.ListenAndServe(":8080", nil)

	ws, _, _ := websocket.DefaultDialer.Dial("ws://127.0.0.1:8080/ws", nil)
	defer ws.Close()

	client := jsonrpc.NewClient(ws.UnderlyingConn())
	defer client.Close()

	req := "test"
	var res string
	client.Call("Test.Echo", &req, &res)

	assert.Equal(t, req, res)
}
