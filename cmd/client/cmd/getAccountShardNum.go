/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

var account *string
var privatekey *string

// getaccountshardnumCmd represents the get account shard number command
var getaccountshardnumCmd = &cobra.Command{
	Use:   "getaccountshardnum",
	Short: "get account shard number with specified account",
	Long: `get account shard number with specified account
	For example:
		client.exe getaccountshardnum --account 0x007d1b1ea335e8e4a74c0be781d828dc7db934b1
		client.exe getaccountshardnum --privatekey 0xa2d0d4176db2ee522ae9d35146cf9b75dab3fa0f308028e96e2821aa882c2ce5`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(*privatekey) > 0 {
			key, err := crypto.LoadECDSAFromString(*privatekey)
			if err != nil {
				fmt.Printf("failed to load the private key: %s\n", err.Error())
				return
			}

			addr := crypto.GetAddress(&key.PublicKey)
			*account = addr.ToHex()
		}

		accountAddress, err := common.HexToAddress(*account)
		if err != nil {
			fmt.Printf("the account is invalid for: %v\n", err)
			return
		}
		shard := accountAddress.Shard()

		fmt.Printf("shard number: %d\n", shard)
	},
}

func init() {
	account = getaccountshardnumCmd.Flags().StringP("account", "", "", "account")
	privatekey = getaccountshardnumCmd.Flags().StringP("privatekey", "", "", "private key")

	rootCmd.AddCommand(getaccountshardnumCmd)
}
