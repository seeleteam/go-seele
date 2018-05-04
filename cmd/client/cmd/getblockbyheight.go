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
	height *int64
	tx     *string
)

// getblockbyheightCmd represents the get block by height command
var getblockbyheightCmd = &cobra.Command{
	Use:   "getblockbyheight",
	Short: "get block info by block height",
	Long: `For example:
	client.exe getblockbyheight --height -1 [-f true] [-a 127.0.0.1:55027]`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		hashRequest := seele.GetBlockByHeightRequest{
			Height: *height,
			FullTx: *tx == "true",
		}
		var result map[string]interface{}
		err = client.Call("seele.GetBlockByHeight", &hashRequest, &result)
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
	rootCmd.AddCommand(getblockbyheightCmd)

	height = getblockbyheightCmd.Flags().Int64("height", -1, "block height")
	getblockbyheightCmd.MarkFlagRequired("height")

	tx = getblockbyheightCmd.Flags().StringP("fulltx", "f", "false", "is add full tx, default is false")
}
