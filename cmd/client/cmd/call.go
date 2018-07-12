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
	"github.com/spf13/cobra"
)

var callParameter = util.TxInfo{}

// callCmd represents the call command
var callCmd = &cobra.Command{
	Use:   "call",
	Short: "call tx",
	Long: `call tx
   For example:
     client.exe call -m 0 -t 0x<public address> -f keyfile --fee 0 --payload 0x<abi bytecode>
	 client.exe call -a 127.0.0.1:8027 -m 0 -t 0x<public address> -f keyfile --fee 0 --payload 0x<abi bytecode>`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Printf("invalid address: %s\n", err.Error())
			return
		}
		defer client.Close()

		pass, err := common.GetPassword()
		if err != nil {
			fmt.Printf("failed to get password %s\n", err.Error())
			return
		}

		key, err := keystore.GetKey(*callParameter.From, pass)
		if err != nil {
			fmt.Printf("invalid sender key file. it should be a private key: %s\n", err.Error())
			return
		}

		txd, ok := util.CheckParameter(callParameter, &key.PrivateKey.PublicKey, client)
		if !ok {
			return
		}

		result, ok := util.Call(client, key.PrivateKey, &txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
		if ok {
			fmt.Println("succeeded to call the tx")
			str, err := json.MarshalIndent(result, "", "\t")
			if err != nil {
				fmt.Println("failed to marshal receipt", err)
				return
			}

			fmt.Println(string(str))
		}
	},
}

func init() {
	rootCmd.AddCommand(callCmd)

	callParameter.To = callCmd.Flags().StringP("to", "t", "", "the contract address")
	callCmd.MarkFlagRequired("to")

	callParameter.Amount = callCmd.Flags().StringP("amount", "m", "", "the amount of the transferred coins")
	callCmd.MarkFlagRequired("amount")

	callParameter.From = callCmd.Flags().StringP("from", "f", "", "key file path of the sender")
	callCmd.MarkFlagRequired("from")

	callParameter.Fee = callCmd.Flags().StringP("fee", "", "", "transaction fee")
	callCmd.MarkFlagRequired("fee")

	callParameter.Payload = callCmd.Flags().StringP("payload", "", "", "transaction payload")
	callCmd.MarkFlagRequired("payload")
}
