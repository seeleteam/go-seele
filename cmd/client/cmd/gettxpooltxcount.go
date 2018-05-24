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

// gettxpooltxcountCmd represents the get tx pool status command
var gettxpooltxcountCmd = &cobra.Command{
	Use:   "gettxpooltxcount",
	Short: "get the number of all processable transactions contained within the transaction pool",
	Long: `For example:
	client.exe gettxpooltxcount`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var status uint64
		err = client.Call("debug.GetTxPoolTxCount", nil, &status)

		if err != nil {
			fmt.Printf("get tx pool status failed %s\n", err.Error())
		}
		fmt.Printf("tx pool status : %d\n", status)
	},
}

func init() {
	rootCmd.AddCommand(gettxpooltxcountCmd)
}
