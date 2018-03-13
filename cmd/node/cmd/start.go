/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/seeleteam/go-seele/node"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the node of seele",
	Long: `usage example:
		node start 
		start a node.`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("start called")

		seeleNode := &node.Node{}
		seeleNode.Start()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
