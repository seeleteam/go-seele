/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"os"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rpcAddr string
var wsAddr string

// rootCmd represents the base command called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "client",
	Short: "rpc client",
	Long:  `rpc client to interact with node process`,
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
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&rpcAddr, "addr", "a", "127.0.0.1:55027", "rpc address")
	rootCmd.PersistentFlags().StringVarP(&wsAddr, "wsaddr", "w", "ws://127.0.0.1:8080/ws", "websocket rpc address")
	rootCmd.AddCommand(util.GetGenerateKeyPairCmd("client"))
	rootCmd.AddCommand(util.GetAccountShardNumCmd())
}

// initConfig reads in the config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using the config file:", viper.ConfigFileUsed())
	}
}
