/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/spf13/cobra"
)

var account *string

// getacountshardnumCmd represents the get account shard number command
var getacountshardnumCmd = &cobra.Command{
	Use:   "getacountshardnum",
	Short: "get account shard number with specified account",
	Long: `get account shard number with specified account
	For example:
		client.exe getacountshardnum --account 0x007d1b1ea335e8e4a74c0be781d828dc7db934b1`,
	Run: func(cmd *cobra.Command, args []string) {
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
	account = getacountshardnumCmd.Flags().StringP("account", "", "", "account")
	rootCmd.AddCommand(getacountshardnumCmd)
}
