/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"net/rpc/jsonrpc"

	"github.com/seeleteam/go-seele/seele"
	"github.com/spf13/cobra"
)

var (
	number *int64
	tx     *string
)

// getblockbynumberCmd represents the get block by number command
var getblockbynumberCmd = &cobra.Command{
	Use:   "getblockbynumber",
	Short: "get block info by block number",
	Long: `For example:
	client.exe getblockbynumber -n -1 [-f true] [-a 127.0.0.1:55027]`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		hashRequest := seele.GetBlockByNumberRequest{
			Number: *number,
			FullTx: *tx == "true",
		}
		var result map[string]interface{}
		err = client.Call("seele.GetBlockByNumber", &hashRequest, &result)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonResult, err := json.Marshal(result)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("block : %s\n", string(jsonResult))
	},
}

func init() {
	rootCmd.AddCommand(getblockbynumberCmd)

	number = getblockbynumberCmd.Flags().Int64P("number", "n", -1, "block number")
	getblockbynumberCmd.MarkFlagRequired("number")

	tx = getblockbynumberCmd.Flags().StringP("fulltx", "f", "false", "is add full tx, default is false")
}
