/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package rpc

import (
	"testing"
)

type WSTest struct{}

func (t *WSTest) Echo(req *string, res *string) error {
	*res = *req
	return nil
}
func Test_Websocket(t *testing.T) {
	handler := NewWsRPCServer()
	rpcServer := handler.GetWsRPCServer()
	err := rpcServer.RegisterName("Test", new(WSTest))
	if err != nil {
		t.Fatalf("Websocket register test failed")
	}
}
