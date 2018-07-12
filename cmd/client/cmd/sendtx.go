/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/seeleteam/go-seele/seele"
	"github.com/spf13/cobra"
)

var parameter = util.TxInfo{}

// sendtxCmd represents the sendtx command
var sendtxCmd = &cobra.Command{
	Use:   "sendtx",
	Short: "send a tx to the miner",
	Long: `send a tx to the miner
  For example:
    client.exe sendtx -m 0 -t 0x<public address> -f keyfile
    client.exe sendtx -a 127.0.0.1:8027 -m 0 -t 0x<public address> -f keyfile `,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Printf("invalid address: %s\n", err.Error())
			return
		}
		defer client.Close()

		pass, err := common.GetPassword()
		if err != nil {
			fmt.Printf("get password failed %s\n", err.Error())
			return
		}

		key, err := keystore.GetKey(*parameter.From, pass)
		if err != nil {
			fmt.Printf("invalid sender key file. it should be a private key: %s\n", err.Error())
			return
		}

		txd, ok := util.CheckParameter(parameter, &key.PrivateKey.PublicKey, client)
		if !ok {
			return
		}

		tx, ok := util.Sendtx(client, key.PrivateKey, &txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
		if ok {
			fmt.Println("adding the tx succeeded.")
			printTx := seele.PrintableOutputTx(tx)
			str, err := json.MarshalIndent(printTx, "", "\t")
			if err != nil {
				fmt.Println("marshal transaction failed ", err)
				return
			}

			fmt.Println(string(str))
		}
	},
}

func init() {
	rootCmd.AddCommand(sendtxCmd)

	parameter.To = sendtxCmd.Flags().StringP("to", "t", "", "public address of the receiver")

	parameter.Amount = sendtxCmd.Flags().StringP("amount", "m", "", "the amount of the transferred coins")
	sendtxCmd.MarkFlagRequired("amount")

	parameter.From = sendtxCmd.Flags().StringP("from", "f", "", "key file path of the sender")
	sendtxCmd.MarkFlagRequired("from")

	parameter.Fee = sendtxCmd.Flags().StringP("fee", "", "", "transaction fee")
	sendtxCmd.MarkFlagRequired("fee")

	parameter.Payload = sendtxCmd.Flags().StringP("payload", "", "", "transaction payload")

	parameter.Nonce = sendtxCmd.Flags().Uint64P("nonce", "", util.DefaultNonce, "nonce of the transaction. If not set, it will get from the node (default 0)")
}
