/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
	"gopkg.in/fatih/set.v0"
)

var num int
var value uint64
var keyFile string
var output string
var shardRange string

var generateKeystoreCmd = &cobra.Command{
	Use:   "genkeys",
	Short: "generate key file list",
	Long: `For example:
	tool.exe genkeys`,
	Run: func(cmd *cobra.Command, args []string) {
		shards := strings.Split(shardRange, ",")
		shardSet := set.New()
		for _, s := range shards {
			i, err := strconv.Atoi(s)
			if err != nil {
				panic(err)
			}

			shardSet.Add(uint(i))
		}

		var results map[common.Address]*big.Int
		results = make(map[common.Address]*big.Int)
		bigValue := big.NewInt(0).SetUint64(value)

		var keyList bytes.Buffer
		for i := 0; i < num; {
			addr, privateKey, err := crypto.GenerateKeyPair()
			if !shardSet.Has(addr.Shard()) {
				continue
			}

			if err != nil {
				panic(err)
			}

			results[*addr] = bigValue

			key := crypto.FromECDSA(privateKey)
			keyList.WriteString(hexutil.BytesToHex(key))
			keyList.WriteString("\n")

			i++
		}

		err := ioutil.WriteFile(keyFile, keyList.Bytes(), os.ModePerm)
		if err != nil {
			panic(fmt.Sprintf("write key file failed %s", err))
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
	generateKeystoreCmd.Flags().StringVarP(&keyFile, "keyfile", "f", "keystore.txt", "key file path")
	generateKeystoreCmd.Flags().StringVarP(&output, "output", "o", "accounts.json", "output address map file path")
	generateKeystoreCmd.Flags().StringVarP(&shardRange, "shards", "", "1;2", "shard range, split by ,")
}
