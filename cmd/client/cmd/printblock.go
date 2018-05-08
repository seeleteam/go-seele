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

var heightPrint *int64

// printblockCmd represents the print block  command
var printblockCmd = &cobra.Command{
	Use:   "printblock",
	Short: "get block pretty printed form by block height",
	Long: `For example:
	client.exe printblock --height -1 [-a 127.0.0.1:55027]`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var result string
		err = client.Call("seele.PrintBlock", &heightPrint, &result)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("block rlp : %s\n", result)
	},
}

func init() {
	rootCmd.AddCommand(printblockCmd)

	heightPrint = printblockCmd.Flags().Int64("height", -1, "block height")
	printblockCmd.MarkFlagRequired("height")
}
