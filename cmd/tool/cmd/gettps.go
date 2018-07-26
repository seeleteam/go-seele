/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"time"

	"github.com/seeleteam/go-seele/seele"
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
			sum := float64(0)
			for _, client := range clientList {
				var tps seele.TpsInfo
				err := client.Call("debug.GetTPS", nil, &tps)
				if err != nil {
					fmt.Println("failed to get tps ", err)
					return
				}

				shard := getShard(client)
				if tps.Duration > 0 {
					t := float64(tps.Count) / float64(tps.Duration)
					fmt.Printf("shard %d: from %d to %d, block number:%d, tx count:%d, interval:%d, tps:%.2f\n", shard, tps.StartHeight,
						tps.EndHeight, tps.EndHeight-tps.StartHeight, tps.Count, tps.Duration, t)
					sum += t
				}
			}

			fmt.Printf("sum tps is %.2f\n", sum)
			time.Sleep(10 * time.Second)
		}
	},
}

func init() {
	rootCmd.AddCommand(gettps)
}
