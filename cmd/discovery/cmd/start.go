/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"net"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/spf13/cobra"
)

var (
	addr          *string //node addr
	bootstrapNode *string //bootstrap node id
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "discovery command for find node and detect node",
	Long: `usage example:
    discovery start 
        start a server, it will generate a node id randomly. The default address is 127.0.0.1:9000
    discovery start -i snode://2aa34f83208861645c9f1b26e4314ced1540788f190564e2bd9594c5da4b68d1e46a8054a590b4a923beaac6c007c120571597586ff099d06e109d7f4769f021@127.0.0.1:9000
        start a server and specific node id.
    discovery start -b snode://2aa34f83208861645c9f1b26e4314ced1540788f190564e2bd9594c5da4b68d1e46a8054a590b4a923beaac6c007c120571597586ff099d06e109d7f4769f021@127.0.0.1:9000 -a "127.0.0.1:9001"
        start a server with a bootstrap node and specific its binding address.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("start called")

		var bootstrap *discovery.Node
		bootstrap = nil
		if *bootstrapNode != "" {
			n, err := discovery.NewNodeFromString(*bootstrapNode)
			if err != nil {
				fmt.Println(err)
				return
			}

			bootstrap = n
		}

		var mynode *discovery.Node
		if *id == "" { // if node id is specified, will ignore address
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

			mynode = discovery.NewNodeWithAddr(*myId, myAddr)
			fmt.Println(mynode.String())
		} else {
			n, err := discovery.NewNodeFromString(*id)
			if err != nil {
				fmt.Println(err)
				return
			}

			mynode = n
		}

		discovery.StartService(mynode.ID, mynode.GetUDPAddr(), bootstrap)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	addr = startCmd.Flags().StringP("addr", "a", "127.0.0.1:9000", "node addr")
	bootstrapNode = startCmd.Flags().StringP("bootstrapNode", "b", "", "bootstrap node id")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
