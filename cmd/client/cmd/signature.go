/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/spf13/cobra"
)

var privateKey *string
var param = util.TxInfo{}

// signCmd represents the sign command
var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "sign the data with your private key",
	Long: `sign the data with your private key
  For example:
    client.exe sign -m 0 -t 0x<public address> -fee 0 -k privatekey
    client.exe sign -a 127.0.0.1:55027 -m 0 -t 0x<public address> -fee 0 -k privatekey`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Printf("invalid address: %s\n", err.Error())
			return
		}
		defer client.Close()

		key, err := crypto.LoadECDSAFromString(*privateKey)
		if err != nil {
			fmt.Printf("load key failed %s\n", err)
			return
		}

		txd, ok := util.CheckParameter(param, &key.PublicKey, client)
		if !ok {
			return
		}

		var tx = types.Transaction{}
		tx.Data = *txd
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

	privateKey = signCmd.Flags().StringP("key", "k", "", "private key")
	signCmd.MarkFlagRequired("key")

	param.To = signCmd.Flags().StringP("to", "t", "", "public address of the receiver")
	signCmd.MarkFlagRequired("to")

	param.Amount = signCmd.Flags().StringP("amount", "m", "", "the amount of the transferred coins")
	signCmd.MarkFlagRequired("amount")

	param.Fee = signCmd.Flags().StringP("fee", "", "", "transaction fee")
	signCmd.MarkFlagRequired("fee")

	param.Payload = signCmd.Flags().StringP("payload", "", "", "transaction payload")
}
