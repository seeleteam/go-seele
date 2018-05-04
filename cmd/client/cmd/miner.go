/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"net/rpc/jsonrpc"

	"github.com/spf13/cobra"
)

var threadsNum *int
var operation *string

// getbalanceCmd represents the getbalance command
var minerCmd = &cobra.Command{
	Use:   "miner",
	Short: "miner actions",
	Long: `For example:
	 client.exe miner -a start
	 client.exe miner -a stop`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var result string
		var input string
		switch *operation {
		case "start":
			err = client.Call("miner.Start", &threadsNum, &result)
			if err != nil {
				fmt.Printf("miner start failed: %s\n", err.Error())
				return
			}
			fmt.Println("miner start")
			return
		case "stop":
			err = client.Call("miner.Stop", &input, &result)
			if err != nil {
				fmt.Printf("miner stop failed: %s\n", err.Error())
				return
			}
			fmt.Println("miner stop")
			return
		default:
			fmt.Println("operation is not defined.")
			return

		}
	},
}

func init() {
	rootCmd.AddCommand(minerCmd)

	threadsNum = minerCmd.Flags().IntP("threads", "t", 0, "threads num of the miner")

	operation = minerCmd.Flags().StringP("operation", "o", "", "operation of the miner, exp[start, stop]")
	minerCmd.MarkFlagRequired("operation")
}
