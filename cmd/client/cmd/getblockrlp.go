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

var heightRlp *int64

// getblockrlpCmd represents the get block rlp command
var getblockrlpCmd = &cobra.Command{
	Use:   "getblockrlp",
	Short: "get block rlp hex by block height",
	Long: `For example:
	client.exe getblockrlp --height -1 [-a 127.0.0.1:55027]`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var result string
		err = client.Call("debug.GetBlockRlp", &heightRlp, &result)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("block rlp : %s\n", result)
	},
}

func init() {
	rootCmd.AddCommand(getblockrlpCmd)

	heightRlp = getblockrlpCmd.Flags().Int64("height", -1, "block height")
	getblockrlpCmd.MarkFlagRequired("height")
}
