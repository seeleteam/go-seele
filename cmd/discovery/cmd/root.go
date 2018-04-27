/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var (
	id *string //specified node id. if set, the node address will be ignored
)

// rootCmd represents the base command called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "discovery",
	Short: "discovery command for finding nodes and detecting nodes",
	Long:  `use "discovery help [<command>]" for detailed usage`,
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

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.discovery.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	id = rootCmd.PersistentFlags().StringP("id", "i", "", "node id")
}

// initConfig reads in the config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use cfgFile as the config file.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		// Search the config file in home directory with name ".discovery" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".discovery")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Using the config file: %s\n", viper.ConfigFileUsed())
	}
}
