/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var gettps = &cobra.Command{
	Use:   "tps",
	Short: "get tps from server list",
	Long: `For example:
		tool.exe tps`,
	Run: func(cmd *cobra.Command, args []string) {
		initClient()

		for {
			sum := uint64(0)
			for _, client := range clientList {
				var tps uint64
				err := client.Call("debug.GetTPS", nil, &tps)
				if err != nil {
					fmt.Println("get tps failed ", err)
				}

				shard := getShard(client)
				sum += tps

				fmt.Printf("shard %d tps %d\n", shard, tps)
			}

			fmt.Printf("sum tps is %d, real tps is %d\n", sum, sum/600)
			time.Sleep(10 * time.Second)
		}
	},
}

func init() {
	rootCmd.AddCommand(gettps)
}
