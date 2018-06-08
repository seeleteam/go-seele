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
			sum := uint64(0)
			for _, client := range clientList {
				var tps seele.TpsInfo
				err := client.Call("debug.GetTPS", nil, &tps)
				if err != nil {
					fmt.Println("get tps failed ", err)
					return
				}

				shard := getShard(client)
				if tps.Duration > 0 {
					fmt.Printf("shard %d: from %d to %d, block number:%d, tx count:%d, interval:%d, tps:%d\n", shard, tps.StartHeight,
						tps.EndHeight, tps.EndHeight-tps.StartHeight, tps.Count, tps.Duration, tps.Count/tps.Duration)
					sum += tps.Count / tps.Duration
				}
			}

			fmt.Printf("sum tps is %d\n", sum)
			time.Sleep(10 * time.Second)
		}
	},
}

func init() {
	rootCmd.AddCommand(gettps)
}
