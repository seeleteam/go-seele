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

var
(
	myport *string
	targetid *string
	targetport *string
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
		fmt.Println(*id)

		discovery.SendPing(*myport, *targetid, *targetport)
	},
}

func init() {
	rootCmd.AddCommand(sendCmd)

	myport = sendCmd.Flags().String("myport", "9001", "my port")
	targetid = sendCmd.Flags().String("targetid", "", "target id")
	targetport = sendCmd.Flags().String("targetport", "9000", "target port")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sendCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sendCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
