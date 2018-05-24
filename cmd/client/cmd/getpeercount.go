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

// getPeerCount represents the peercount command
var getPeerCount = &cobra.Command{
	Use:   "peercount",
	Short: "get count of connected peers",
	Long: `get count of connected peers
	 For example:
		 client.exe peercount [-a 127.0.0.1:55027]`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Printf("Failed to connect to the node %s, error:%s\n", rpcAddr, err.Error())
			return
		}
		defer client.Close()

		var peerCount int
		err = client.Call("network.GetPeerCount", nil, &peerCount)
		if err != nil {
			fmt.Printf("get peer count failed %s\n", err.Error())
		} else {
			fmt.Printf("peer count: %d\n", peerCount)
		}
	},
}

func init() {
	rootCmd.AddCommand(getPeerCount)
}
