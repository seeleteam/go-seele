/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package comm

import (
	"fmt"

	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

// generateKeyPairCmd represents the generateKeyPair command

func GetGenerateKeyPairCmd(name string) (cmds *cobra.Command) {
	var generateKeyPairCmd = &cobra.Command{
		Use:   "generatekeypair",
		Short: "generate a key pair",
		Long:  "generate a key pair and print them with hex values\n For example:\n" + name + " generateKeyPair",
		Run: func(cmd *cobra.Command, args []string) {
			publicKey, privateKey, err := crypto.GenerateKeyPair()
			if err != nil {
				fmt.Printf("generating the key pair failed: %s\n", err.Error())
			}

			fmt.Printf("public key: %s\n", publicKey.ToHex())
			fmt.Printf("private key: %s\n", hexutil.BytesToHex(crypto.FromECDSA(privateKey)))
		},
	}
	return generateKeyPairCmd
}
