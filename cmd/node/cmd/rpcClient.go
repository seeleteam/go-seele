/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"net/rpc/jsonrpc"

	"github.com/BurntSushi/toml"
	"github.com/seeleteam/go-seele/common"
	"github.com/spf13/cobra"
)

var clientConfigFile *string

// ClientConfig is the rpc client config
type ClientConfig struct {
	RPCAddr string
}

// rpcClientCmd the rpc test client to node.
var rpcClientCmd = &cobra.Command{
	Use:   "rpcClient",
	Short: "test the rpc of node",
	Long: `usage example:
		node.exe rpcClient -c cmd\rpcClient.toml
		start a rpc client with config file to test rpc server.`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("testrpc called")
		clientConfig := new(ClientConfig)
		_, err := toml.DecodeFile(*clientConfigFile, clientConfig)
		if err != nil {
			fmt.Println(err)
			return
		}

		client, err := jsonrpc.Dial("tcp", clientConfig.RPCAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		input := 1
		addr := new(common.Address)
		err = client.Call("seele.Coinbase", input, addr)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("get address: %v\n", *addr)

		return
	},
}

func init() {
	rootCmd.AddCommand(rpcClientCmd)

	clientConfigFile = rpcClientCmd.Flags().StringP("config", "c", "", "seele node config file (required)")
	rpcClientCmd.MarkFlagRequired("config")

}
