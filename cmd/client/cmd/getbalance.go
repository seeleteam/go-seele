/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/spf13/cobra"
)

var address *common.Address
var accountStr string

func getClientCmd(use, short, long string, handler func(client *rpc.Client)) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Run: func(cmd *cobra.Command, args []string) {
			client, err := rpc.Dial("tcp", rpcAddr)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			defer client.Close()

			if accountStr == "" {
				address = nil
			} else {
				result, err := common.HexToAddress(accountStr)
				if err != nil {
					fmt.Printf("invalid account address:%s, %s\n", accountStr, err.Error())
					return
				}

				address = &result
			}

			handler(client)
		},
	}
}

func init() {
	// getbalanceCmd represents the getbalance command
	var getbalanceCmd = getClientCmd("getbalance", "get the balance of an account",
		`For example:
		client.exe getbalance`,
		func(client *rpc.Client) {
			amount := big.NewInt(0)
			err := client.Call("seele.GetBalance", &address, amount)
			if err != nil {
				fmt.Printf("failed to get the balance: %s\n", err)
				return
			}

			if address == nil {
				fmt.Printf("no account is provided. the coinbase balance: %s\n", common.BigToDecimal(amount))
			} else {
				fmt.Printf("Account: %s\nBalance: %s\n", address.ToHex(), common.BigToDecimal(amount))
			}
		})

	var getnonceCmd = getClientCmd("getnonce", "get nonce of an account",
		`For example: client.exe getnonce`,
		func(client *rpc.Client) {
			if address == nil {
				fmt.Println("must specific address")
				return
			}

			util.GetNonce(client, *address)
		})

	rootCmd.AddCommand(getbalanceCmd)
	rootCmd.AddCommand(getnonceCmd)

	getbalanceCmd.Flags().StringVarP(&accountStr, "account", "t", "", "account address")
	getnonceCmd.Flags().StringVarP(&accountStr, "account", "t", "", "account address")
}
