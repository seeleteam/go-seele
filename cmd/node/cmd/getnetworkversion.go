/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"net/rpc/jsonrpc"

	"github.com/spf13/cobra"
)

// getNetworkVersion represents the networkversion command
var getNetworkVersion = &cobra.Command{
	Use:   "networkversion",
	Short: "get current protocol version",
	Long: `get current protocol version
	  For example:
		  node.exe networkversion -a 127.0.0.1:55027`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var networkVersion int
		err = client.Call("network.GetNetworkVersion", nil, &networkVersion)
		if err != nil {
			fmt.Printf("get network version failed %s\n", err.Error())
		}

		fmt.Printf("network version: %d\n", networkVersion)
	},
}

func init() {
	rootCmd.AddCommand(getNetworkVersion)
}
