/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/seeleteam/go-seele/rpc"
	"github.com/spf13/cobra"
)

var threadsNum *int
var start bool
var status bool
var stop bool
var gethashrate bool

// getbalanceCmd represents the getbalance command
var minerCmd = &cobra.Command{
	Use:   "miner",
	Short: "miner actions",
	Long: `For example:
	client.exe miner --start [-t <miner threads num>]
	client.exe miner --status
	client.exe miner --stop
	client.exe miner --gethashrate`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		var result string
		var input string
		if start {
			err = client.Call("miner.Start", &threadsNum, &result)
			if err != nil {
				fmt.Printf("failed to start miner: %s\n", err.Error())
				return
			}
			fmt.Println("miner started successfully")
		} else if status {
			err = client.Call("miner.Status", &input, &result)
			if err != nil {
				fmt.Printf("failed to get miner status: %s\n", err.Error())
				return
			}
			fmt.Printf("miner status is: %s\n", result)
		} else if stop {
			err = client.Call("miner.Stop", &input, &result)
			if err != nil {
				fmt.Printf("failed to stop miner: %s\n", err.Error())
				return
			}
			fmt.Println("miner stopped successfully")
		} else if gethashrate {
			var hashrate uint64
			err = client.Call("miner.Hashrate", &input, &hashrate)
			if err != nil {
				fmt.Printf("failed to get miner's hashrate: %s\n", err.Error())
				return
			}
			fmt.Printf("miner hashrate is: %d\n", hashrate)
		} else {
			fmt.Println("command param is not defined.")
		}
	},
}

func init() {
	rootCmd.AddCommand(minerCmd)

	threadsNum = minerCmd.Flags().IntP("threads", "t", 0, "threads num of the miner")

	minerCmd.Flags().BoolVar(&start, "start", false, "start miner")
	minerCmd.Flags().BoolVar(&status, "status", false, "view miner status")
	minerCmd.Flags().BoolVar(&stop, "stop", false, "stop miner")
	minerCmd.Flags().BoolVar(&gethashrate, "gethashrate", false, "get hashrate of the miner")
}
