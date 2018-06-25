/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

var keyStr *string
var keyFile *string

// savekey represents the savekey command
var savekey = &cobra.Command{
	Use:   "savekey",
	Short: "save the key",
	Long: `save the private key
    For example:
		client.exe savekey -k 0x<privatekey>`,
	Run: func(cmd *cobra.Command, args []string) {
		privateKey, err := crypto.LoadECDSAFromString(*keyStr)
		if err != nil {
			fmt.Printf("invalid key: %s\n", err.Error())
			return
		}

		if keyFile == nil || *keyFile == "" {
			fmt.Printf("invalid key file path\n")
			return
		}

		pass, err := common.SetPassword()
		if err != nil {
			fmt.Printf("get password err %s\n", err.Error())
			return
		}

		key := keystore.Key{
			Address:    *crypto.GetAddress(&privateKey.PublicKey),
			PrivateKey: privateKey,
		}

		err = keystore.StoreKey(*keyFile, pass, &key)
		if err != nil {
			fmt.Printf("failed to store the key file %s, %s\n", *keyFile, err.Error())
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(savekey)

	keyStr = savekey.Flags().StringP("key", "k", "", "private key")
	savekey.MarkFlagRequired("key")

	keyFile = savekey.Flags().StringP("file", "f", ".keystore", "key file")
}
