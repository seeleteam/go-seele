/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/seeleteam/go-seele/cmd/comm"
)

var rpcAddr string

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
	rootCmd.PersistentFlags().StringVarP(&rpcAddr, "addr", "a", "127.0.0.1:55027", "rpc address")
	rootCmd.AddCommand(comm.GetGenerateKeyPairCmd("node"))
}
