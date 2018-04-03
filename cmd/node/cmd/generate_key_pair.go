/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

// generateKeyPairCmd represents the generateKeyPair command
var generateKeyPairCmd = &cobra.Command{
	Use:   "generatekeypair",
	Short: "generate key pair",
	Long: `generate key pair, print them with hex value
		For example:
			node.exe generateKeyPair`,
	Run: func(cmd *cobra.Command, args []string) {
		publicKey, privateKey, err := crypto.GenerateKeyPair()
		if err != nil {
			fmt.Printf("get error: %s", err.Error())
		}

		fmt.Printf("public key: %s\n", publicKey.ToHex())
		fmt.Printf("private key: %s\n", hexutil.BytesToHex(crypto.FromECDSA(privateKey)))
	},
}

func init() {
	rootCmd.AddCommand(generateKeyPairCmd)
}
