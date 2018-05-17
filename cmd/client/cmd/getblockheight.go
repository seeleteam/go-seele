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

// getblockheightCmd represents the get block height command
var getblockheightCmd = &cobra.Command{
	Use:   "getblockheight",
	Short: "get block height of the chain head",
	Long: `For example:
	client.exe getblockheight`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var height uint64
		err = client.Call("seele.GetBlockHeight", nil, &height)

		if err != nil {
			fmt.Printf("get block height failed %s\n", err.Error())
		}
		fmt.Printf("head block height is %d\n", height)
	},
}

func init() {
	rootCmd.AddCommand(getblockheightCmd)
}
