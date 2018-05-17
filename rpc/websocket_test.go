/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package rpc

import (
	"net/http"
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
	handler := NewWsRPCServer()
	rpcServer := handler.GetWsRPCServer()
	rpcServer.RegisterName("Test", new(WSTest))
	http.HandleFunc("/test", handler.ServeWS)
	go http.ListenAndServe("127.0.0.1:12315", nil)

	ws, _, _ := websocket.DefaultDialer.Dial("ws://127.0.0.1:12315/test", nil)
	defer ws.Close()

	client := NewClient(ws.UnderlyingConn())
	defer client.Close()

	req := "test"
	var res string
	client.Call("Test.Echo", &req, &res)

	assert.Equal(t, req, res)
}
