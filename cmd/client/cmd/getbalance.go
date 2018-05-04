/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"math/big"
	"net/rpc/jsonrpc"

	"github.com/seeleteam/go-seele/common"
	"github.com/spf13/cobra"
)

var (
	account *string
)

// getbalanceCmd represents the getbalance command
var getbalanceCmd = &cobra.Command{
	Use:   "getbalance",
	Short: "get the balance of an account",
	Long: `For example:
	client.exe getbalance`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer client.Close()

		var address *common.Address
		if account == nil || *account == "" {
			address = nil
		} else {
			result, err := common.HexToAddress(*account)
			if err != nil {
				fmt.Printf("invalid account address: %s\n", err.Error())
				return
			}

			address = &result
		}

		amount := big.NewInt(0)
		err = client.Call("seele.GetBalance", &address, amount)
		if err != nil {
			fmt.Printf("getting the balance failed: %s\n", err.Error())
		}

		if address == nil {
			fmt.Printf("no account is provided. the coinbase balance: %s\n", amount)
		} else {
			fmt.Printf("Account: %s\nBalance: %s\n", address.ToHex(), amount)
		}
	},
}

func init() {
	rootCmd.AddCommand(getbalanceCmd)

	account = getbalanceCmd.Flags().StringP("account", "t", "", "account address")
}
