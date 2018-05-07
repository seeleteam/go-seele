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

// gettxpoolstatusCmd represents the get tx pool status command
var gettxpoolstatusCmd = &cobra.Command{
	Use:   "gettxpoolstatus",
	Short: "get the number of all processable transactions contained within the transaction pool",
	Long: `For example:
	client.exe gettxpoolstatus`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var status uint64
		err = client.Call("seele.GetTxPoolStatus", nil, &status)

		if err != nil {
			fmt.Printf("get tx pool status failed %s\n", err.Error())
		}
		fmt.Printf("tx pool status : %d\n", status)
	},
}

func init() {
	rootCmd.AddCommand(gettxpoolstatusCmd)
}
