/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"path/filepath"
	"time"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/seeleteam/go-seele/seele"
	"github.com/spf13/cobra"
)

var keyFolder string

type balance struct {
	address    *common.Address
	privateKey *ecdsa.PrivateKey
	amount     int
	shard      uint
	nonce      uint64
}

var sendTxCmd = &cobra.Command{
	Use:   "sendtx",
	Short: "send tx peroidly",
	Long: `For example:
	tool.exe sendtx`,
	Run: func(cmd *cobra.Command, args []string) {
		initClient()
		balanceList := initAccount()

		// send tx periodically
		for {
			rand.Seed(time.Now().UnixNano())
			bIndex := rand.Intn(len(balanceList))
			b := balanceList[bIndex]

			//update balance from current node
			newAmount, ok := getbalance(*b.address)
			if ok {
				b.amount = newAmount
			}

			if b.amount == 0 {
				continue
			}

			amount := rand.Intn(b.amount) // for test, amount will always keep in int value.
			addr, privateKey := crypto.MustGenerateShardKeyPair(b.address.Shard())
			newBalance := &balance{
				address:    addr,
				privateKey: privateKey,
				amount:     amount,
				shard:      addr.Shard(),
				nonce:      0,
			}

			value := big.NewInt(int64(amount))
			value.Mul(value, common.SeeleToFan)
			// update nonce
			b.nonce++
			client := getRandClient()
			if util.Sendtx(client, b.privateKey, addr, value, big.NewInt(0), b.nonce, nil) {
				// update balance by transaction amount
				b.amount -= amount
				if b.amount == 0 {
					balanceList = append(balanceList[:bIndex], balanceList[bIndex+1:]...)
				}

				if newBalance.amount > 0 {
					balanceList = append(balanceList, newBalance)
				}

				fmt.Printf("%s balance changed from %d to %d\n", b.address.ToHex(), b.amount+amount, b.amount)
				fmt.Printf("%s balance changed from %d to %d\n", newBalance.address.ToHex(), 0, newBalance.amount)
				fmt.Printf("%s nonce is %d", b.address.ToHex(), b.nonce)
				fmt.Println()
			}

			time.Sleep(time.Second)
		}
	},
}

func getRandClient() *rpc.Client {
	if len(clientList) == 0 {
		panic("no client found")
	}

	index := rand.Intn(len(clientList))

	count := 0
	for _, v := range clientList {
		if count == index {
			return v
		}

		count++
	}

	return nil
}

func initAccount() []*balance {
	balanceList := make([]*balance, 0)

	// init file list from key store folder
	keyFiles := make([]string, 0)
	files, err := ioutil.ReadDir(keyFolder)
	if err != nil {
		fmt.Println("read directory err ", err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		path := filepath.Join(keyFolder, f.Name())
		keyFiles = append(keyFiles, path)

		fmt.Println("find key file ", path)
	}

	// init balance and nonce
	for _, f := range keyFiles {
		key, err := keystore.GetKey(f, password)
		if err != nil {
			fmt.Println("get private key failed ", err)
		}

		addr := crypto.GetAddress(&key.PrivateKey.PublicKey)
		amount, ok := getbalance(*addr)
		if !ok {
			continue
		}

		b := &balance{
			address:    addr,
			privateKey: key.PrivateKey,
			amount:     amount,
			shard:      addr.Shard(),
		}

		fmt.Printf("%s balance is %d\n", b.address.ToHex(), b.amount)

		if b.amount > 0 {
			b.nonce = getNonce(*b.address)
			balanceList = append(balanceList, b)
		}
	}

	return balanceList
}

func getbalance(address common.Address) (int, bool) {
	client := getClient(address)

	amount := big.NewInt(0)
	err := client.Call("seele.GetBalance", &address, amount)
	if err != nil {
		fmt.Printf("getting the balance failed: %s\n", err.Error())
		return 0, false
	}

	return int(amount.Div(amount, common.SeeleToFan).Uint64()), true
}

func getClient(address common.Address) *rpc.Client {
	shard := address.Shard()
	client := clientList[shard]
	if client == nil {
		panic(fmt.Sprintf("not found client in shard %d", shard))
	}

	return client
}

func getNonce(address common.Address) uint64 {
	client := getClient(address)

	return util.GetNonce(client, address)
}

func getShard(client *rpc.Client) uint {
	var info seele.MinerInfo
	err := client.Call("seele.GetInfo", nil, &info)
	if err != nil {
		fmt.Printf("getting the balance failed: %s\n", err.Error())
		return 0
	}

	return info.Coinbase.Shard() // @TODO need refine this code, get shard info straight
}

func init() {
	rootCmd.AddCommand(sendTxCmd)

	sendTxCmd.Flags().StringVarP(&keyFolder, "keyfolder", "f", "..\\client\\keyfile", "key file folder")
}
