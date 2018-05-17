/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/seeleteam/go-seele/rpc"
	"github.com/seeleteam/go-seele/seele"
	"github.com/spf13/cobra"
)

// getinfo represents the getinfo command
var getinfo = &cobra.Command{
	Use:   "getinfo",
	Short: "get the miner info",
	Long: `get the miner info
    For example:
		client.exe getinfo -a 127.0.0.1:55027`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer client.Close()

		var info seele.MinerInfo
		err = client.Call("seele.GetInfo", nil, &info)
		if err != nil {
			fmt.Printf("getting the miner info failed: %s\n", err.Error())
		}

		fmt.Printf("coinbase address: %s\n", info.Coinbase.ToHex())
		fmt.Printf("current block height: %d\n", info.CurrentBlockHeight)
		fmt.Printf("current block header hash: %s\n", info.HeaderHash.ToHex())
	},
}

func init() {
	rootCmd.AddCommand(getinfo)
}
