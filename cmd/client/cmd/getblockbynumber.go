/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"net/rpc/jsonrpc"

	"github.com/seeleteam/go-seele/core/types"
	"github.com/spf13/cobra"
)

var number *string

// getblockbynumberCmd represents the get block by number command
var getblockbynumberCmd = &cobra.Command{
	Use:   "getblockbynumber",
	Short: "get block info by block number",
	Long: `For example:
	client.exe getblockbynumber -n -1`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var block types.Block
		err = client.Call("seele.GetBlockByNumber", &number, &block)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("block hash: %s\n", block.HeaderHash.ToHex())
		fmt.Printf("block height: %d\n", block.Header.Height)
		fmt.Printf("block creator: %s\n", block.Header.Creator.ToHex())
		fmt.Printf("block previousBlockHash: %s\n", block.Header.PreviousBlockHash.ToHex())
		fmt.Printf("block stateHash: %s\n", block.Header.StateHash.ToHex())
		fmt.Printf("block txHash: %s\n", block.Header.TxHash.ToHex())
		fmt.Printf("block nonce: %d\n", block.Header.Nonce)
		fmt.Printf("block difficulty: %d\n", block.Header.Difficulty)
		fmt.Printf("block createTimestamp: %d\n", block.Header.CreateTimestamp)
		for index, transaction := range block.Transactions {
			fmt.Printf("block transaction %d hash: %s\n", index, transaction.Hash.ToHex())
			fmt.Printf("block transaction %d from: %s\n", index, transaction.Data.From.ToHex())
			fmt.Printf("block transaction %d to: %s\n", index, transaction.Data.To.ToHex())
			fmt.Printf("block transaction %d amount: %d\n", index, transaction.Data.Amount)
			fmt.Printf("block transaction %d payload: %d\n", index, transaction.Data.Payload)
			fmt.Printf("block transaction %d nonce: %d\n", index, transaction.Data.AccountNonce)
			fmt.Printf("block transaction %d timestamp: %d\n", index, transaction.Data.Timestamp)
		}
	},
}

func init() {
	rootCmd.AddCommand(getblockbynumberCmd)

	number = getblockbynumberCmd.Flags().StringP("number", "n", "", "block number")
	getblockbynumberCmd.MarkFlagRequired("number")
}
