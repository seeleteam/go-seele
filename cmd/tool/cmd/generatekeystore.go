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
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
	"gopkg.in/fatih/set.v0"
)

var (
	num         int
	value       uint64
	keyFile     string
	receivers   string
	output      string
	shard       int
	accountFile string
)

// KeyInfo information of account key
type KeyInfo struct {
	addr       *common.Address
	privateKey string
}

// AccountInfo users info
type AccountInfo struct {
	Account    string `json:"Account"`
	PrivateKey string `json:"PrivateKey"`
}

var generateKeystoreCmd = &cobra.Command{
	Use:   "genkeys",
	Short: "generate key file list",
	Long: `For example:
	tool.exe genkeys`,
	Run: func(cmd *cobra.Command, args []string) {
		shardSet := set.New()
		for i := 1; i <= shard; i++ {
			shardSet.Add(uint(i))
		}

		wg := sync.WaitGroup{}
		infos := make([]*KeyInfo, num)
		for i := 0; i < threads; i++ {
			wg.Add(1)

			go func(start int) {
				defer wg.Done()
				for j := start; j < num; {
					addr, privateKey, err := crypto.GenerateKeyPair()
					if err != nil {
						panic(err)
					}

					if !shardSet.Has(addr.Shard()) {
						continue
					}

					infos[j] = &KeyInfo{
						addr:       addr,
						privateKey: hexutil.BytesToHex(crypto.FromECDSA(privateKey)),
					}

					j += threads
				}
			}(i)
		}

		wg.Wait()

		fmt.Println("key generate success")
		var results map[common.Address]*big.Int
		results = make(map[common.Address]*big.Int)
		bigValue := big.NewInt(0).SetUint64(value)

		var keyList bytes.Buffer
		users := make(map[uint][]AccountInfo)
		for _, info := range infos {
			users[info.addr.Shard()] = append(users[info.addr.Shard()], AccountInfo{Account: info.addr.String(), PrivateKey: info.privateKey})
		}

		data, err := json.MarshalIndent(users, "", "\t")
		if err != nil {
			panic(fmt.Sprintf("Failed to marshal infos %s", err))
		}

		err = ioutil.WriteFile(accountFile, data, os.ModePerm)

		for i := 0; i < num; i++ {
			results[*infos[i].addr] = bigValue

			keyList.WriteString(infos[i].privateKey)
			keyList.WriteString("\r\n")
		}

		err = ioutil.WriteFile(keyFile, keyList.Bytes(), os.ModePerm)
		if err != nil {
			panic(fmt.Sprintf("failed to write key file %s", err))
		}

		str, err := json.MarshalIndent(results, "", "\t")
		if err != nil {
			panic(err)
		}

		err = ioutil.WriteFile(output, str, os.ModePerm)
		if err != nil {
			fmt.Println("failed to write file ", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(generateKeystoreCmd)

	generateKeystoreCmd.Flags().IntVarP(&num, "num", "n", 10, "number of generate key files")
	generateKeystoreCmd.Flags().Uint64VarP(&value, "value", "v", 1000000000000, "init account value of these keys")
	generateKeystoreCmd.Flags().StringVarP(&keyFile, "keyfile", "f", "keystore.txt", "key file path")
	generateKeystoreCmd.Flags().StringVarP(&output, "output", "o", "accounts.json", "output address map file path")
	generateKeystoreCmd.Flags().StringVarP(&accountFile, "userFile", "u", "userFile.json", "file of private key and account")
	generateKeystoreCmd.Flags().IntVarP(&shard, "shard", "", 1, "shard number, it will generate key in [1:shard]")
	generateKeystoreCmd.Flags().IntVarP(&threads, "threads", "t", 1, "threads to generate keys")
}
