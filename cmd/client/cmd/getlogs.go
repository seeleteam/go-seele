/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/seeleteam/go-seele/seele"

	"github.com/seeleteam/go-seele/rpc"
	"github.com/spf13/cobra"
)

// GetLogsRequest request parameter for client command line
type GetLogsRequest struct {
	Height          *int64
	ContractAddress *string
	Topics          *string
}

var getlogsParameter GetLogsRequest

// getlogsCmd represents the getlogs command
var getlogsCmd = &cobra.Command{
	Use:   "getlogs",
	Short: "get logs of the block",
	Long: `get logs of the block
   For example:
     client.exe getlogs -h <block height> -t 0x<contract address> -n 0x<event name hash>
	 client.exe getlogs -a 127.0.0.1:8027 -h <block height> -t 0x<contract address> -n 0x<event name hash>`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Printf("invalid address: %s\n", err.Error())
			return
		}
		defer client.Close()

		request := seele.GetLogsRequest{
			Height:          *getlogsParameter.Height,
			ContractAddress: *getlogsParameter.ContractAddress,
			Topics:          *getlogsParameter.Topics,
		}
		result := make([]map[string]interface{}, 0)
		if err = client.Call("seele.GetLogs", &request, &result); err != nil {
			fmt.Printf("failed to get logs: %s\n", err.Error())
			return
		}

		str, err := json.MarshalIndent(result, "", "\t")
		if err != nil {
			fmt.Printf("failed to marshal result: %s\n", err.Error())
			return
		}

		fmt.Println(string(str))
	},
}

func init() {
	rootCmd.AddCommand(getlogsCmd)

	getlogsParameter.Height = getlogsCmd.Flags().Int64P("height", "", -1, "block height (default -1)")

	getlogsParameter.ContractAddress = getlogsCmd.Flags().StringP("to", "t", "", "the contract address")
	getlogsCmd.MarkFlagRequired("to")

	getlogsParameter.Topics = getlogsCmd.Flags().StringP("topic", "", "", "event name hash")
	getlogsCmd.MarkFlagRequired("topic")
}
