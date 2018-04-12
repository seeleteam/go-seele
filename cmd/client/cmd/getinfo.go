/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"net/rpc/jsonrpc"

	"github.com/seeleteam/go-seele/seele"
	"github.com/spf13/cobra"
)

// getinfo represents the getcoinbase command
var getinfo = &cobra.Command{
	Use:   "getinfo",
	Short: "get miner info",
	Long: `get miner info
    For example:
		client.exe getinfo -a 127.0.0.1:55027`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var info seele.MinerInfo
		err = client.Call("seele.GetInfo", nil, &info)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("coinbase address: %s\n", info.Coinbase.ToHex())
		fmt.Printf("current block height: %d\n", info.CurrentBlockHeight)
		fmt.Printf("current block header hash: %s\n", info.HeaderHash.ToHex())
	},
}

func init() {
	rootCmd.AddCommand(getinfo)
}
