/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"net/rpc/jsonrpc"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/spf13/cobra"
)

// gettxpoolcontentCmd represents the get tx pool content command
var gettxpoolcontentCmd = &cobra.Command{
	Use:   "gettxpoolcontent",
	Short: "get content of the tx pool",
	Long: `For example:
	client.exe gettxpoolcontent`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var result map[common.Address][]*types.Transaction
		err = client.Call("seele.GetTxPoolContent", nil, &result)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonResult, err := json.MarshalIndent(&result, "", "\t")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("tx pool content :\n", string(jsonResult))
	},
}

func init() {
	rootCmd.AddCommand(gettxpoolcontentCmd)
}
