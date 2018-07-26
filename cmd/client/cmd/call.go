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
	Short: "call a contract",
	Long: `call a contract
   For example:
     client.exe call -t 0x<public address> -f keyfile --payload 0x<abi bytecode>
	 client.exe call -a 127.0.0.1:8027 -t 0x<public address> -f keyfile --payload 0x<abi bytecode>`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Printf("invalid address: %s\n", err.Error())
			return
		}
		defer client.Close()

		pass, err := common.GetPassword()
		if err != nil {
			fmt.Printf("failed to get password: %s\n", err.Error())
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
			fmt.Printf("invalid payload, %s\n", err.Error())
			return
		}

		amount := big.NewInt(0)
		fee := big.NewInt(1)
		nonce := uint64(1)
		result, ok := util.Call(client, key.PrivateKey, &contractAddr, amount, fee, nonce, payload)
		if ok {
			fmt.Println("contract called successfully")
			str, err := json.MarshalIndent(result, "", "\t")
			if err != nil {
				fmt.Printf("failed to marshal receipt: %s\n", err.Error())
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
