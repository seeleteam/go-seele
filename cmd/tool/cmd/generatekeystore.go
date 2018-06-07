/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

var num int
var value uint64
var folder string
var output string

var generateKeystoreCmd = &cobra.Command{
	Use:   "genkeys",
	Short: "generate key file list",
	Long: `For example:
	tool.exe genkeys`,
	Run: func(cmd *cobra.Command, args []string) {
		var results map[common.Address]*big.Int
		results = make(map[common.Address]*big.Int)
		bigValue := big.NewInt(0).SetUint64(value)

		for i := 0; i < num; i++ {
			addr, privateKey, err := crypto.GenerateKeyPair()
			if err != nil {
				panic(err)
			}

			results[*addr] = bigValue

			key := keystore.Key{
				Address:    *addr,
				PrivateKey: privateKey,
			}

			fileName := fmt.Sprintf("shard%d-%s", addr.Shard(), addr.ToHex())
			filePath := filepath.Join(folder, fileName)
			keystore.StoreKey(filePath, password, &key)
		}

		str, err := json.MarshalIndent(results, "", "\t")
		if err != nil {
			panic(err)
		}

		err = ioutil.WriteFile(output, str, os.ModePerm)
		if err != nil {
			fmt.Println("write file failed ", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(generateKeystoreCmd)

	generateKeystoreCmd.Flags().IntVarP(&num, "num", "n", 10, "number of generate key files")
	generateKeystoreCmd.Flags().Uint64VarP(&value, "value", "v", 1000000000000, "init account value of these keys")
	generateKeystoreCmd.Flags().StringVarP(&folder, "folder", "f", "keystore", "key file folder")
	generateKeystoreCmd.Flags().StringVarP(&output, "output", "o", "out.txt", "output address map file path")
}
