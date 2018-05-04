/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"math/big"
	"net/rpc/jsonrpc"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

type txInfo struct {
	amount *uint64 // amount specifies the coin amount to be transferred
	to     *string // to is the public address of the receiver
	from   *string // from is the key file path of the sender
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
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Printf("invalid address: %s\n", err.Error())
			return
		}
		defer client.Close()

		toAddr, err := common.HexToAddress(*parameter.to)
		if err != nil {
			fmt.Printf("invalid receiver address: %s\n", err.Error())
			return
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

		from, err := crypto.GetAddress(key.PrivateKey)
		if err != nil {
			fmt.Printf("generating the sender address failed: %s\n", err.Error())
			return
		}

		var nonce uint64
		err = client.Call("seele.GetAccountNonce", &from, &nonce)
		if err != nil {
			fmt.Printf("getting the sender account nonce failed: %s\n", err.Error())
			return
		}

		fmt.Printf("got the sender account nonce: %d\n", nonce)

		amount := big.NewInt(0).SetUint64(*parameter.amount)
		tx := types.NewTransaction(*from, toAddr, amount, nonce)
		tx.Sign(key.PrivateKey)

		var result bool
		err = client.Call("seele.AddTx", &tx, &result)
		if !result || err != nil {
			fmt.Printf("adding the tx failed: %s\n", err.Error())
			return
		}

		fmt.Println("adding the tx succeeded.")
	},
}

func init() {
	rootCmd.AddCommand(sendtxCmd)

	parameter.to = sendtxCmd.Flags().StringP("to", "t", "", "public address of the receiver")
	sendtxCmd.MarkFlagRequired("to")

	parameter.amount = sendtxCmd.Flags().Uint64P("amount", "m", 0, "the amount of the transferred coins")
	sendtxCmd.MarkFlagRequired("amount")

	parameter.from = sendtxCmd.Flags().StringP("from", "f", "", "key file path of the sender")
	sendtxCmd.MarkFlagRequired("from")
}
