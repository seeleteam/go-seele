/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(dbCmd)
}

var dbCmd = &cobra.Command{
	Use:   "initdb",
	Short: "init or clean db",
	Long:  "create a new db or clean the data of existing db for simulator",
	Run: func(cmd *cobra.Command, args []string) {
		if err := os.RemoveAll(defaultDir); err != nil {
			fmt.Println("Failed to init db:", err.Error())
			return
		}

		fmt.Println("db initiated successfully:", defaultDir)
	},
}
