/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

var (
	privateKey *string
)

// validatekeyCmd represents the validatekey command
var validatekeyCmd = &cobra.Command{
	Use:   "validatekey",
	Short: "validate the private key and generate its public key",
	Long: `For example:
			node.exe validatekey`,
	Run: func(cmd *cobra.Command, args []string) {
		key, err := crypto.LoadECDSAFromString(*privateKey)
		if err != nil {
			fmt.Printf("failed to load the private key: %s\n", err.Error())
			return
		}

		addr := crypto.GetAddress(&key.PublicKey)

		fmt.Printf("public key: %s\n", addr.Hex())
	},
}

func init() {
	rootCmd.AddCommand(validatekeyCmd)

	privateKey = validatekeyCmd.Flags().StringP("key", "k", "", "private key")
	validatekeyCmd.MarkFlagRequired("key")
}
