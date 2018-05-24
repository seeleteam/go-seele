/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/seeleteam/go-seele/rpc"

	"github.com/spf13/cobra"
)

// getNetworkVersion represents the networkversion command
var getNetworkVersion = &cobra.Command{
	Use:   "networkversion",
	Short: "get current network version",
	Long: `get current network version
	  For example:
		  client.exe networkversion [-a 127.0.0.1:55027]`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Printf("Failed to connect to the node %s, error:%s\n", rpcAddr, err.Error())
			return
		}
		defer client.Close()

		var networkVersion int
		err = client.Call("network.GetNetworkVersion", nil, &networkVersion)
		if err != nil {
			fmt.Printf("get network version failed %s\n", err.Error())
		} else {
			fmt.Printf("network version: %d\n", networkVersion)
		}
	},
}

func init() {
	rootCmd.AddCommand(getNetworkVersion)
}
