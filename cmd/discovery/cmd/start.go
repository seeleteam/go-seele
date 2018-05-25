/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"net"
	"sync"

	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/spf13/cobra"
)

var (
	addr          *string //node address
	bootstrapNode *string //bootstrap node id
	shard         *uint
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start command for starting node discovery",
	Long: `usage example:
    discovery start 
        start a server which will generate a node id randomly. The default address is 127.0.0.1:9000
    discovery start -i snode://2aa34f83208861645c9f1b26e4314ced1540788f190564e2bd9594c5da4b68d1e46a8054a590b4a923beaac6c007c120571597586ff099d06e109d7f4769f021@127.0.0.1:9000[0]
        start a server with the specified node id.
    discovery start -b snode://2aa34f83208861645c9f1b26e4314ced1540788f190564e2bd9594c5da4b68d1e46a8054a590b4a923beaac6c007c120571597586ff099d06e109d7f4769f021@127.0.0.1:9000[0] -a "127.0.0.1:9001"
        start a server with a bootstrap node and specify its binding address.`,
	Run: func(cmd *cobra.Command, args []string) {
		bootstrap := make([]*discovery.Node, 0)
		if *bootstrapNode != "" {
			n, err := discovery.NewNodeFromString(*bootstrapNode)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			bootstrap = append(bootstrap, n)
		}

		var mynode *discovery.Node
		if *id == "" { // ignore the address if node id is specified
			myAddr, err := net.ResolveUDPAddr("udp", *addr)
			if err != nil {
				fmt.Printf("invalid address: %s\n", err.Error())
				return
			}

			myId, err := crypto.GenerateRandomAddress()
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			mynode = discovery.NewNodeWithAddr(*myId, myAddr, *shard)
			fmt.Println(mynode.String())
		} else {
			n, err := discovery.NewNodeFromString(*id)
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			mynode = n
		}

		discovery.StartService(mynode.ID, mynode.GetUDPAddr(), bootstrap, *shard)

		wg := sync.WaitGroup{}
		wg.Add(1)
		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	addr = startCmd.Flags().StringP("addr", "a", "127.0.0.1:9000", "node addr")
	bootstrapNode = startCmd.Flags().StringP("bootstrapNode", "b", "", "bootstrap node id")
	shard = startCmd.Flags().UintP("shard", "s", 1, "shard number")
}
