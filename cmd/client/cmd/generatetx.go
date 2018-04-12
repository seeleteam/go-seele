/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"net/rpc/jsonrpc"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/spf13/cobra"
	"github.com/seeleteam/go-seele/crypto"
	"math/big"
)

type txInfo struct {
	amount *uint64
	to     *string
	from *string
}

var parameter = txInfo{}

// generatetxCmd represents the sendtx command
var generatetxCmd = &cobra.Command{
	Use:   "generatetx",
	Short: "generate tx to miner",
	Long: `generate tx to miner
  For example:
    client.exe sendtx -m 0 -t 0x1cba7cc4097c34ef9d90c0bf1fa9babd7e2fb26db7b49d7b1eb8f580726e3a99d3aec263fc8de535e74a79138622d320b3765b0a75fabd084985c456c6fe65bb -f 
    client.exe sendtx -a 127.0.0.1:55027 -m 0 -t 0x1cba7cc4097c34ef9d90c0bf1fa9babd7e2fb26db7b49d7b1eb8f580726e3a99d3aec263fc8de535e74a79138622d320b3765b0a75fabd084985c456c6fe65bb`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Printf("invalid address. %s\n", err.Error())
			return
		}
		defer client.Close()

		toAddr, err := common.HexToAddress(*parameter.to)
		if err != nil {
			fmt.Printf("invalid to address. %s\n", err.Error())
			return
		}

		privateKey, err := crypto.LoadECDSAFromString(*parameter.from)
		if err != nil {
			fmt.Printf("invalid from key. it should be a private key. %s\n", err.Error())
			return
		}

		from, err := crypto.GetAddress(privateKey)
		if err != nil {
			fmt.Printf("generate address failed, %s\n", err.Error())
			return
		}

		var nonce uint64
		err = client.Call("seele.GetAccountNonce", &from, &nonce)
		if err != nil {
			fmt.Printf("get account nonce failed %s\n", err.Error())
			return
		}

		fmt.Printf("get account nonce %d\n", nonce)

		amount := big.NewInt(0).SetUint64(*parameter.amount)
		tx := types.NewTransaction(*from, toAddr, amount, nonce)
		tx.Sign(privateKey)

		var result bool
		err = client.Call("seele.GenerateTx", &tx, &result)
		if !result || err != nil {
			fmt.Printf("add tx failed. %s\n", err.Error())
			return
		}

		fmt.Println("add tx successful.")
	},
}

func init() {
	rootCmd.AddCommand(generatetxCmd)

	parameter.to = generatetxCmd.Flags().StringP("to", "t", "", "to user's public key")
	generatetxCmd.MarkFlagRequired("to")

	parameter.amount = generatetxCmd.Flags().Uint64P("amount", "m", 0, "the number of the transaction value")
	generatetxCmd.MarkFlagRequired("amount")

	parameter.from = generatetxCmd.Flags().StringP("from", "f", "", "from user's private key")
	generatetxCmd.MarkFlagRequired("from")
}
