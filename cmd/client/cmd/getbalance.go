/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"net/rpc/jsonrpc"
	"math/big"
)

// getbalanceCmd represents the getbalance command
var getbalanceCmd = &cobra.Command{
	Use:   "getbalance",
	Short: "get balance of the coinbase",
	Long: `For example:
	client.exe getbalance`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		amount := big.NewInt(0)
		err = client.Call("seele.GetBalance", nil, amount)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("coinbase balance: %s", amount)
	},
}

func init() {
	rootCmd.AddCommand(getbalanceCmd)
}
