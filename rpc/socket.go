/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package rpc

import (
	"context"
	"net"
)

// DialTCP create client with tcp connection
func DialTCP(ctx context.Context, endpoint string) (*Client, error) {
	return newClient(ctx, func(ctx context.Context) (net.Conn, error) {
		return net.Dial("tcp", endpoint)
	})
}
