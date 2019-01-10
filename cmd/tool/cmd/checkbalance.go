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

	"github.com/ethereum/go-ethereum/params"
	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/spf13/cobra"
)

var (
	senderAccounts string
	// senders address
	sendersAddress map[uint][]KeyInfo
)

// currently only work for one shard
var checkBalanceCmd = &cobra.Command{
	Use:   "checkbalance",
	Short: "check the balance consistency of the sender addresses and the receiver addresses",
	Long: `For example:
	 tool.exe checkbalance`,
	Run: func(cmd *cobra.Command, args []string) {
		initClient()
		accounts, err := LoadAccountConfig(senderAccounts)
		if err != nil {
			panic(fmt.Sprintf("failed to read sender accounts %s", err))
		}

		data, err := ioutil.ReadFile(receivers)
		if err != nil {
			panic(fmt.Sprintf("failed to read receivers file %s", err))
		}

		if err = json.Unmarshal(data, &receiversAddress); err != nil {
			panic(fmt.Sprintf("Failed to unmarshal %s", err))
		}

		totalSendersAmount := big.NewInt(0)
		for account := range accounts {
			amount, ok := getFullBalance(account, "", -1)
			if !ok {
				panic(fmt.Sprintf("Failed to get balance"))
			}
			totalSendersAmount.Add(totalSendersAmount, amount)
		}

		totalReceiversAmount := big.NewInt(0)
		initToAccount()
		for shardNum := range receiversAddress {
			data := receiversAddress[shardNum]
			for index := range data {
				addr := data[index].Addr
				amount, ok := getFullBalance(*addr, "", -1)
				if !ok {
					panic(fmt.Sprintf("Failed to get balance"))
				}
				totalReceiversAmount.Add(totalReceiversAmount, amount)
			}
		}

		var height uint64
		var counter uint64
		var txCount int
		txCount = 0
		var blockTxCount int
		for clientIndex := range clientList {
			if err := clientList[clientIndex].Call(&height, "seele_getBlockHeight"); err != nil {
				panic(fmt.Sprintf("failed to get the block height: %s", err))
			}
			fmt.Printf("block height %d\n", height)
			counter = 1
			// get the tx count up to current block height
			for counter <= height {
				blockTxCount = 0
				if err := clientList[clientIndex].Call(&blockTxCount, "txpool_getBlockTransactionCount", "", counter); err != nil {
					panic(fmt.Sprintf("failed to get the block tx count: %s\n", err))
				}
				txCount += blockTxCount - 1
				counter++
			}
		}

		fmt.Printf("sender balance %d\n", totalSendersAmount)
		fmt.Printf("receiver balance  %d\n", totalReceiversAmount)
		fmt.Printf("tx count %d\n", txCount)
		fmt.Printf("tx fee %d\n", txCount*int(params.TxGas))
		totalAmount := big.NewInt(0)
		totalAmount.Add(totalAmount, totalSendersAmount)
		totalAmount.Add(totalAmount, totalReceiversAmount)
		totalAmount.Add(totalAmount, big.NewInt(int64(txCount*int(params.TxGas))))
		fmt.Printf("total amount %d\n", totalAmount)

	},
}

func LoadAccountConfig(account string) (map[common.Address]*big.Int, error) {
	result := make(map[common.Address]*big.Int)
	if account == "" {
		return result, nil
	}

	buff, err := ioutil.ReadFile(account)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(buff, &result)
	return result, err
}

func getFullBalance(address common.Address, hexHash string, height int64) (*big.Int, bool) {
	client := getClient(address)

	var result api.GetBalanceResponse
	if err := client.Call(&result, "seele_getBalance", address, hexHash, height); err != nil {
		panic(fmt.Sprintf("failed to get the balance: %s\n", err))
	}

	return result.Balance, true
}

func init() {
	rootCmd.AddCommand(checkBalanceCmd)

	checkBalanceCmd.Flags().StringVarP(&senderAccounts, "sender", "a", "", "sender address file")
	checkBalanceCmd.Flags().StringVarP(&receivers, "receiver", "r", "", "receiver address file")
}
