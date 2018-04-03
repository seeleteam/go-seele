/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"net/rpc/jsonrpc"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/seele"
	"github.com/spf13/cobra"
)

var (
	amount *uint64
	to     *string
)

// sendtxCmd represents the sendtx command
var sendtxCmd = &cobra.Command{
	Use:   "sendtx",
	Short: "send tx to miner",
	Long: `send tx to miner
  For example:
    client.exe sendtx -m 0 -t 0x1cba7cc4097c34ef9d90c0bf1fa9babd7e2fb26db7b49d7b1eb8f580726e3a99d3aec263fc8de535e74a79138622d320b3765b0a75fabd084985c456c6fe65bb
    client.exe sendtx -a 127.0.0.1:55027 -m 0 -t 0x1cba7cc4097c34ef9d90c0bf1fa9babd7e2fb26db7b49d7b1eb8f580726e3a99d3aec263fc8de535e74a79138622d320b3765b0a75fabd084985c456c6fe65bb`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Printf("invalid address. %s\n", err.Error())
			return
		}
		defer client.Close()

		toAddr, err := common.HexToAddress(*to)
		if err != nil {
			fmt.Printf("invalid to address. %s\n", err.Error())
			return
		}

		rpcArgs := seele.AddTxArgs{
			To:     toAddr,
			Amount: *amount,
		}

		var result bool
		err = client.Call("seele.AddTx", rpcArgs, &result)
		if !result || err != nil {
			fmt.Printf("add tx failed. %s\n", err.Error())
			return
		}

		fmt.Println("add tx successful.")
	},
}

func init() {
	rootCmd.AddCommand(sendtxCmd)

	to = sendtxCmd.Flags().StringP("to", "t", "", "to user's public key")
	sendtxCmd.MarkFlagRequired("to")

	amount = sendtxCmd.Flags().Uint64P("amount", "m", 0, "the number of the transaction value")
	sendtxCmd.MarkFlagRequired("amount")
}
