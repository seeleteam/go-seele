/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"github.com/seeleteam/go-seele/common"
	"net"

	"github.com/seeleteam/go-seele/p2p/discovery"

	"github.com/spf13/cobra"
)

var
(
	addr          *string
	bootstrapNode *string
)

// sendCmd represents the send command
var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("send called")

		nodeId, err := discovery.NewNodeFromString(*bootstrapNode)
		if err != nil {
			fmt.Println(err)
			return
		}

		myAddr, err := net.ResolveUDPAddr("udp", *addr)
		if err != nil {
			fmt.Println("invalid address", err.Error())
			return
		}

		myId, err := common.GenerateRandomAddress()
		if err != nil {
			fmt.Println(err)
			return
		}

		mynode := discovery.NewNodeWithAddr(*myId, myAddr)
		fmt.Println(mynode.String())

		discovery.SendPing(*myId, myAddr, nodeId.ID, nodeId.GetUDPAddr())
	},
}


func init() {
	rootCmd.AddCommand(sendCmd)

	addr = sendCmd.Flags().StringP("addr", "a", ":9001", "node addr")
	bootstrapNode = sendCmd.Flags().StringP("bootstrapNode", "b", "", "bootstrap node id")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sendCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sendCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
