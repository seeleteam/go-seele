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

// getinfo represents the getinfo command
var getPeerCount = &cobra.Command{
	Use:   "peercount",
	Short: "get count of connected peers",
	Long: `get count of connected peers
	 For example:
		 node.exe peercount -a 127.0.0.1:55027`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var peerCount int
		err = client.Call("network.GetPeerCount", nil, &peerCount)
		if err != nil {
			fmt.Printf("get peer count failed %s\n", err.Error())
		}

		fmt.Printf("peer count: %d\n", peerCount)
	},
}

func init() {
	rootCmd.AddCommand(getPeerCount)
}
