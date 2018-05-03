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

// getblocknumberCmd represents the get block number command
var getblocknumberCmd = &cobra.Command{
	Use:   "getblocknumber",
	Short: "get block number of the chain head",
	Long: `For example:
	client.exe getblocknumberCmd`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var number uint64
		err = client.Call("seele.GetBlockNumber", nil, &number)

		if err != nil {
			fmt.Printf("get block number failed %s\n", err.Error())
		}
		fmt.Printf("head block number is %d\n", number)
	},
}

func init() {
	rootCmd.AddCommand(getblocknumberCmd)
}
