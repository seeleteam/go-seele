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

// getcoinbaseCmd represents the getcoinbase command
var getcoinbaseCmd = &cobra.Command{
	Use:   "getcoinbase",
	Short: "get coinbase address",
	Long: `get coinbase address
    For example:
		client.exe getcoinbase -a 127.0.0.1:55027`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		addr := new(common.Address)
		err = client.Call("seele.Coinbase", nil, addr)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("coinbase address: %v\n", addr.ToHex())
	},
}

func init() {
	rootCmd.AddCommand(getcoinbaseCmd)
}
