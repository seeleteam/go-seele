/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/seeleteam/go-seele/rpc"
	"github.com/spf13/cobra"
)

var (
	// serveList is the servers that provide rpc service
	serveList string

	// shard -> client
	clientList map[uint]*rpc.Client

	// threads the thread to send tx
	threads int
)

// rootCmd represents the base command called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "node",
	Short: "node command for starting a node",
	Long:  `use "node help [<command>]" for detailed usage`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&serveList, "server", "s", "127.0.0.1:8027", "server list for requesting and submit, split by ,")
}

func initClient() {
	addrs := strings.Split(serveList, ",")
	clientList = make(map[uint]*rpc.Client, 0)

	for _, addr := range addrs {
		client, err := rpc.DialTCP(context.Background(), addr)
		if err != nil {
			panic(fmt.Sprintf("dial failed %s for server %s", err, addr))
		}

		shard := getShard(client)
		clientList[shard] = client
	}
}
