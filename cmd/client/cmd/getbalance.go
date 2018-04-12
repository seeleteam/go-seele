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
	"github.com/seeleteam/go-seele/common"
)

var (
	account *string
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

		var address common.Address
		if account == nil || *account == "" {
			address = common.Address{}
		} else {
			address, err = common.HexToAddress(*account)
			if err != nil {
				fmt.Printf("invalid account address. %s\n", err.Error())
				return
			}
		}

		amount := big.NewInt(0)
		err = client.Call("seele.GetBalance", &address, amount)
		if err != nil {
			fmt.Printf("get balance failed %s\n", err.Error())
		}

		fmt.Printf("account balance: %s", amount)
	},
}

func init() {
	rootCmd.AddCommand(getbalanceCmd)

	account = getbalanceCmd.Flags().StringP("account", "t", "", "account address")
}
