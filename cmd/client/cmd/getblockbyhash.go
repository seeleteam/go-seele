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
	hashHex *string
	fullTx  *bool
)

// getblockbyhashCmd represents the get block by hash command
var getblockbyhashCmd = &cobra.Command{
	Use:   "getblockbyhash",
	Short: "get block info by block hash",
	Long: `For example:
	client.exe getblockbyhash --hash 0x0000009721cf7bb5859f1a0ced952fcf71929ff8382db6ef20041ed441d5f92f [-f=true] [-a 127.0.0.1:55027]`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		hashRequest := seele.GetBlockByHashRequest{
			HashHex: *hashHex,
			FullTx:  *fullTx,
		}
		var result map[string]interface{}
		err = client.Call("seele.GetBlockByHash", &hashRequest, &result)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonResult, err := json.MarshalIndent(&result, "", "\t")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("block :\n", string(jsonResult))
	},
}

func init() {
	rootCmd.AddCommand(getblockbyhashCmd)

	hashHex = getblockbyhashCmd.Flags().String("hash", "", "block hash")
	getblockbyhashCmd.MarkFlagRequired("hash")

	fullTx = getblockbyhashCmd.Flags().BoolP("fulltx", "f", false, "is add full tx, default is false")
}
