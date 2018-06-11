/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/seeleteam/go-seele/rpc"
	"github.com/spf13/cobra"
)

var print bool

// gettxpoolcontentCmd represents the get tx pool content command
var gettxpoolcontentCmd = &cobra.Command{
	Use:   "gettxpoolcontent",
	Short: "get content of the tx pool",
	Long: `For example:
	client.exe gettxpoolcontent`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var result map[string][]map[string]interface{}
		err = client.Call("debug.GetTxPoolContent", nil, &result)
		if err != nil {
			fmt.Println(err)
			return
		}

		if print {
			jsonResult, err := json.MarshalIndent(&result, "", "\t")
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("tx pool content :\n", string(jsonResult))
		}

		sum := 0
		for _, value := range result {
			sum += len(value)
		}
		fmt.Printf("tx pool total number: %d\n", sum)
	},
}

func init() {
	rootCmd.AddCommand(gettxpoolcontentCmd)

	gettxpoolcontentCmd.Flags().BoolVarP(&print, "print", "p", false, "whether print out the tx pool content")
}
