/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/spf13/cobra"
)

type txInfo struct {
	amount  *string // amount specifies the coin amount to be transferred
	to      *string // to is the public address of the receiver
	from    *string // from is the key file path of the sender
	fee     *string // transaction fee
	payload *string // transaction payload in hex format
}

var parameter = txInfo{}

// sendtxCmd represents the sendtx command
var sendtxCmd = &cobra.Command{
	Use:   "sendtx",
	Short: "send a tx to the miner",
	Long: `send a tx to the miner
  For example:
    client.exe sendtx -m 0 -t 0x<public address> -f keyfile
    client.exe sendtx -a 127.0.0.1:55027 -m 0 -t 0x<public address> -f keyfile `,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Printf("invalid address: %s\n", err.Error())
			return
		}
		defer client.Close()

		var toAddr common.Address
		if len(*parameter.to) > 0 {
			if toAddr, err = common.HexToAddress(*parameter.to); err != nil {
				fmt.Printf("invalid receiver address: %s\n", err.Error())
				return
			}
		}

		pass, err := common.GetPassword()
		if err != nil {
			fmt.Printf("get password failed %s\n", err.Error())
			return
		}

		key, err := keystore.GetKey(*parameter.from, pass)
		if err != nil {
			fmt.Printf("invalid sender key file. it should be a private key: %s\n", err.Error())
			return
		}

		amount, ok := big.NewInt(0).SetString(*parameter.amount, 10)
		if !ok {
			fmt.Println("invalid amount value")
			return
		}

		fee, ok := big.NewInt(0).SetString(*parameter.fee, 10)
		if !ok {
			fmt.Println("invalid fee value")
			return
		}

		var nonce uint64
		fromAddr := crypto.GetAddress(&key.PrivateKey.PublicKey)
		err = client.Call("seele.GetAccountNonce", fromAddr, &nonce)
		if err != nil {
			fmt.Printf("getting the sender account nonce failed: %s\n", err.Error())
			return
		}
		fmt.Printf("got the sender account %s nonce: %d\n", fromAddr.ToHex(), nonce)

		payload := []byte(nil)
		if len(*parameter.payload) > 0 {
			if payload, err = hexutil.HexToBytes(*parameter.payload); err != nil {
				fmt.Println("invalid payload,", err.Error())
				return
			}
		}

		util.Sendtx(client, key.PrivateKey, &toAddr, amount, fee, nonce, payload)
	},
}

func init() {
	rootCmd.AddCommand(sendtxCmd)

	parameter.to = sendtxCmd.Flags().StringP("to", "t", "", "public address of the receiver")

	parameter.amount = sendtxCmd.Flags().StringP("amount", "m", "", "the amount of the transferred coins")
	sendtxCmd.MarkFlagRequired("amount")

	parameter.from = sendtxCmd.Flags().StringP("from", "f", "", "key file path of the sender")
	sendtxCmd.MarkFlagRequired("from")

	parameter.fee = sendtxCmd.Flags().StringP("fee", "", "", "transaction fee")
	sendtxCmd.MarkFlagRequired("fee")

	parameter.payload = sendtxCmd.Flags().StringP("payload", "", "", "transaction payload")
}
