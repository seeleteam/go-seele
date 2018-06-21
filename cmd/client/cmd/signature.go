/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/spf13/cobra"
)

var data *string
var fileFullName *string

// signatureCmd represents the signature command
var signatureCmd = &cobra.Command{
	Use:   "signature",
	Short: "sign the data with your private key",
	Long: `sign the data with your private key
  For example:
    client.exe signature -d datafile -f keyfile
    client.exe signature -a 127.0.0.1:55027 -d datafile -f keyfile`,
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

		pass, err := common.GetPassword()
		if err != nil {
			fmt.Printf("get password failed %s\n", err.Error())
			return
		}

		key, err := keystore.GetKey(*fileFullName, pass)
		if err != nil {
			fmt.Printf("invalid sender key file. it should be a private key: %s\n", err.Error())
			return
		}

		fromAddr := crypto.GetAddress(&key.PrivateKey.PublicKey)
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
		tx.Sign(key.PrivateKey)

		databytes, err := json.MarshalIndent(tx, "", "")
		if err != nil {
			fmt.Printf("json marshl failed: %s\n", err.Error())
			return
		}
		err = ioutil.WriteFile(*data, databytes, os.ModeAppend)
		if err != nil {
			fmt.Printf("write data file failed: %s\n", err.Error())
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(signatureCmd)

	data = signatureCmd.Flags().StringP("data", "d", "", "the transaction data file path that needs to be signed")
	signatureCmd.MarkFlagRequired("data")

	fileFullName = signatureCmd.Flags().StringP("from", "f", "", "key file path of the sender")
	signatureCmd.MarkFlagRequired("from")
}
