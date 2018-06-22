/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/spf13/cobra"
)

var data *string
var privateKey *string

// signCmd represents the sign command
var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "sign the data with your private key",
	Long: `sign the data with your private key
  For example:
    client.exe sign -d datafile -k privatekey
    client.exe sign -a 127.0.0.1:55027 -d datafile -k privatekey
	the datafile like:
	{	"From": "0x02235268262b72978c20eec2be8244b61dd5a0f1",
		"To": "0x2a87b6504cd00af95a83b9887112016a2a991cf1",
		"Amount": 10, "Fee": 1
	}`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Printf("invalid address: %s\n", err.Error())
			return
		}
		defer client.Close()

		var txd = types.TransactionData{}

		content, err := ioutil.ReadFile(*data)
		if err != nil {
			fmt.Printf("read data file failed: %s\n", err.Error())
			return
		}
		err = json.Unmarshal(content, &txd)
		if err != nil {
			fmt.Printf("data parsing error: %s\n", err.Error())
			return
		}

		key, err := crypto.LoadECDSAFromString(*privateKey)
		if err != nil {
			fmt.Sprintf("load key failed %s", err)
			return
		}

		fromAddr := crypto.GetAddress(&key.PublicKey)
		if txd.From == common.EmptyAddress {
			txd.From = *fromAddr
		}

		if txd.AccountNonce == 0 {
			var nonce uint64
			err = client.Call("seele.GetAccountNonce", fromAddr, &nonce)
			if err != nil {
				fmt.Printf("getting the sender account nonce failed: %s\n", err.Error())
				return
			}
			fmt.Printf("got the sender account %s nonce: %d\n", fromAddr.ToHex(), nonce)
			txd.AccountNonce = nonce + 1
		}

		var tx = types.Transaction{}
		tx.Data = txd
		tx.Sign(key)

		databytes, err := json.MarshalIndent(tx, "\t", "")
		if err != nil {
			fmt.Printf("json marshl failed: %s\n", err.Error())
			return
		}

		fmt.Printf("out: %v\n", string(databytes))
	},
}

func init() {
	rootCmd.AddCommand(signCmd)

	data = signCmd.Flags().StringP("data", "d", "", "the transaction data file path, it's a json file, have four variable; From: account of payment, To: account to be credited, Amount: transfer amount, Fee: tip")
	signCmd.MarkFlagRequired("data")

	privateKey = signCmd.Flags().StringP("key", "k", "", "private key")
	signCmd.MarkFlagRequired("key")
}
