/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package util

import (
	"fmt"
	"github.com/seeleteam/go-seele/common"
	"github.com/spf13/cobra"
)

// GetAccountShardNumCmd represents the get account shard number command
func GetAccountShardNumCmd() (cmds *cobra.Command) {
	var account *string

	var getAccountShardNumCmd = &cobra.Command{
		Use:   "getAccountShardNum",
		Short: "getAccountShardNum account with specified account",
		Long:  `getAccountShardNum account with specified account
		For example:
			client.exe getAccountShardNum --account 0x007d1b1ea335e8e4a74c0be781d828dc7db934b1`,
		Run: func(cmd *cobra.Command, args []string) {
			defer func() {
				if err := recover(); err != nil {
					fmt.Printf("the account is invalid for: %v\n", err)
					return
				}
			}()

			accountAddress := common.HexMustToAddres(*account)
			shard := accountAddress.Shard()

			fmt.Printf("shard number:  %d\n", shard)
		},
	}

	account = getAccountShardNumCmd.Flags().StringP("account", "", "", "account")

	return getAccountShardNumCmd
}
