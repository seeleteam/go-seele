/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"net/rpc/jsonrpc"

	"github.com/seeleteam/go-seele/common"
	"github.com/spf13/cobra"
)

var rpcServerAddr *string

// rpcClientCmd the rpc test client to node.
var rpcClientCmd = &cobra.Command{
	Use:   "rpcClient",
	Short: "test the rpc of node",
	Long: `usage example:
		node.exe rpcClient -a 127.0.0.1:55027
		start a rpc client to test rpc server.`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("testrpc called")

		client, err := jsonrpc.Dial("tcp", *rpcServerAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		input := 1
		addr := new(common.Address)
		err = client.Call("seele.Coinbase", input, addr)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("get address: %v\n", *addr)

		return
	},
}

func init() {
	rootCmd.AddCommand(rpcClientCmd)

	rpcServerAddr = rpcClientCmd.Flags().StringP("address", "a", "", "seele node rpc start address (Ip:Port)")
	rpcClientCmd.MarkFlagRequired("address")
}
