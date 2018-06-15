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
	"strings"
	"time"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/seeleteam/go-seele/seele"
	"github.com/spf13/cobra"
)

var tps int

type balance struct {
	address    *common.Address
	privateKey *ecdsa.PrivateKey
	amount     int
	shard      uint
	nonce      uint64
	tx         *common.Hash
}

var sendTxCmd = &cobra.Command{
	Use:   "sendtx",
	Short: "send tx peroidly",
	Long: `For example:
	tool.exe sendtx`,
	Run: func(cmd *cobra.Command, args []string) {
		initClient()
		balanceList := initAccount()

		var confirmTime = 3 * time.Minute
		count := 0
		tpsStartTime := time.Now()
		toConfirmBalanceList := make(map[time.Time][]*balance)
		newBalanceList := make([]*balance, 0)
		// send tx periodically
		for {
			nextBalanceList := make([]*balance, 0)
			for _, b := range balanceList {
				newBalance := send(b)

				if b.amount > 0 {
					nextBalanceList = append(nextBalanceList, b)
				}

				if newBalance.amount > 0 {
					newBalanceList = append(newBalanceList, newBalance)
				}
				count++

				if count == tps {
					fmt.Println("send txs ", count)
					elapse := time.Now().Sub(tpsStartTime)
					if elapse < time.Second {
						time.Sleep(time.Second - elapse)
					}

					if len(newBalanceList) > 0 {
						toConfirmBalanceList[time.Now()] = newBalanceList
						newBalanceList = make([]*balance, 0)
					}

					count = 0
					tpsStartTime = time.Now()

					for _, value := range toConfirmBalanceList {
						checkTxExist(value)
					}
				}
			}

			balanceList = nextBalanceList
			for key, value := range toConfirmBalanceList {
				duration := time.Now().Sub(key)
				if duration > confirmTime {
					included, pending := getIncludedAndPendingBalance(value)
					balanceList = append(balanceList, included...)
					if len(pending) == 0 {
						delete(toConfirmBalanceList, key)
					} else {
						toConfirmBalanceList[key] = pending
					}

					fmt.Printf("add confirmed balance %d, pending %d\n", len(included), len(pending))
				}
			}
		}
	},
}

func checkTxExist(balances []*balance) {
	for _, b := range balances {
		if b.tx == nil {
			continue
		}

		result := getTx(*b.address, *b.tx)
		if len(result) > 0 {
			//fmt.Printf("got tx success %s from %s nonce %.0f status %s amount %.0f\n", b.tx.ToHex(), result["from"],
			//	result["accountNonce"], result["status"], result["amount"])
		}
	}
}

func getIncludedAndPendingBalance(balances []*balance) ([]*balance, []*balance) {
	include := make([]*balance, 0)
	pending := make([]*balance, 0)
	for _, b := range balances {
		if b.tx == nil {
			continue
		}

		result := getTx(*b.address, *b.tx)
		if len(result) > 0 {
			if result["status"] == "block" {
				include = append(include, b)
			} else if result["status"] == "pool" {
				pending = append(pending, b)
			}
		}
	}

	return include, pending
}

func getTx(address common.Address, hash common.Hash) map[string]interface{} {
	client := getClient(address)
	var result map[string]interface{}
	addrStr := hash.ToHex()
	err := client.Call("txpool.GetTransactionByHash", &addrStr, &result)
	if err != nil {
		fmt.Println("get tx failed ", err, " tx hash ", hash.ToHex())
		return result
	}

	return result
}

func send(b *balance) *balance {
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
	tx, ok := util.Sendtx(client, b.privateKey, addr, value, big.NewInt(0), b.nonce, nil)
	if ok {
		// update balance by transaction amount
		b.amount -= amount
		newBalance.tx = &tx.Hash
	}

	return newBalance
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

	keys, err := ioutil.ReadFile(keyFile)
	if err != nil {
		panic(fmt.Sprintf("read key file failed %s", err))
	}

	keyList := strings.Split(string(keys), "\n")

	// init balance and nonce
	for _, hex := range keyList {
		if hex == "" {
			continue
		}

		key, err := crypto.LoadECDSAFromString(hex)
		if err != nil {
			panic(fmt.Sprintf("load key failed %s", err))
		}

		addr := crypto.GetAddress(&key.PublicKey)
		// skip address that don't find the same shard client
		if _, ok := clientList[addr.Shard()]; !ok {
			continue
		}

		amount, ok := getBalance(*addr)
		if !ok {
			continue
		}

		b := &balance{
			address:    addr,
			privateKey: key,
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

func getBalance(address common.Address) (int, bool) {
	client := getClient(address)

	amount := big.NewInt(0)
	err := client.Call("seele.GetBalance", &address, amount)
	if err != nil {
		panic(fmt.Sprintf("getting the balance failed: %s\n", err.Error()))
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
		panic(fmt.Sprintf("getting the balance failed: %s\n", err.Error()))
		return 0
	}

	return info.Coinbase.Shard() // @TODO need refine this code, get shard info straight
}

func init() {
	rootCmd.AddCommand(sendTxCmd)

	sendTxCmd.Flags().StringVarP(&keyFile, "keyfile", "f", "keystore.txt", "key store file")
	sendTxCmd.Flags().IntVarP(&tps, "tps", "", 3, "target tps to send transaction")
}
