/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
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

		contractAddr, err := common.HexToAddress(*callParameter.To)
		if err != nil {
			fmt.Printf("invalid contract address: %s\n", err.Error())
			return
		}

		payload, err := hexutil.HexToBytes(*callParameter.Payload)
		if err != nil {
			fmt.Println("invalid payload,", err.Error())
			return
		}

		result, ok := util.Call(client, key.PrivateKey, &contractAddr, big.NewInt(0), big.NewInt(0), 1, payload)
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

	callParameter.From = callCmd.Flags().StringP("from", "f", "", "key file path of the sender")
	callCmd.MarkFlagRequired("from")

	callParameter.Payload = callCmd.Flags().StringP("payload", "", "", "transaction payload")
	callCmd.MarkFlagRequired("payload")
}
