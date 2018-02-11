/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/spf13/cobra"
)

// listenCmd represents the listen command
var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("listen called")

		// use a fixed id for test
		id := "snode://c03ff3c956d0a320b153a097c3d04efa488d43d7d7e05a44791492c9979ff558f9956c0a6b0c414783476f02ad8557349d35ba9373dadfa9a7a44fd88328189f@:9000"

		node, err := discovery.NewNodeFromString(id)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(node.String())

		discovery.StartService(node.ID, node.GetUDPAddr())
	},
}

func init() {
	rootCmd.AddCommand(listenCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listenCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listenCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
