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

var hash *string

// getblockbyhashCmd represents the get block by hash command
var getblockbyhashCmd = &cobra.Command{
	Use:   "getblockbyhash",
	Short: "get block info by block hash",
	Long: `For example:
	client.exe getblockbyhash -h 0x0000009721cf7bb5859f1a0ced952fcf71929ff8382db6ef20041ed441d5f92f`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var block types.Block
		err = client.Call("seele.GetBlockByHash", &hash, &block)
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
	rootCmd.AddCommand(getblockbyhashCmd)

	hash = getblockbyhashCmd.Flags().StringP("hash", "s", "", "block hash")
	getblockbyhashCmd.MarkFlagRequired("hash")
}
