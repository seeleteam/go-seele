/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package util

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

// GetGenerateKeyPairCmd represents the generateKeyPair command
func GetGenerateKeyPairCmd(name string) (cmds *cobra.Command) {
	var shard *uint

	var generateKeyPairCmd = &cobra.Command{
		Use:   "key",
		Short: "generate a key pair with specified shard number",
		Long:  "generate a key pair and print them with hex values\n For example:\n" + name + " key --shard 1",
		Run: func(cmd *cobra.Command, args []string) {
			var publicKey *common.Address
			var privateKey *ecdsa.PrivateKey
			var err error
			if *shard > common.ShardCount {
				fmt.Printf("not supported shard number, shard number should be [0, %d]\n", common.ShardCount)
				return
			} else if *shard == 0 {
				publicKey, privateKey, err = crypto.GenerateKeyPair()
				if err != nil {
					fmt.Printf("failed to generate the key pair: %s\n", err.Error())
				}
			} else {
				publicKey, privateKey = crypto.MustGenerateShardKeyPair(*shard)
			}

			fmt.Printf("public key:  %s\n", publicKey.ToHex())
			fmt.Printf("private key: %s\n", hexutil.BytesToHex(crypto.FromECDSA(privateKey)))
		},
	}

	shard = generateKeyPairCmd.Flags().UintP("shard", "", 0, "shard number")

	return generateKeyPairCmd
}
