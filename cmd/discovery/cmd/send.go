/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"

	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/seeleteam/go-seele/cmd/utils"

	"github.com/spf13/cobra"
)

var
(
	port          *string
	bootstrapNode *string
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

		nodeId, err := utils.NewNodeId(*bootstrapNode)
		if err != nil {
			fmt.Println(err)
			return
		}

		discovery.SendPing(*port, nodeId.Address, nodeId.GetUDPAddr())
	},
}

func getRandomNodeID() common.Address {
	keypair, err := crypto.GenerateKey()
	if err != nil {
		log.Info(err.Error())
	}

	buff := crypto.FromECDSAPub(&keypair.PublicKey)

	id, err := common.NewAddress(buff[1:])
	if err != nil {
		log.Fatal(err.Error())
	}

	return id
}


func init() {
	rootCmd.AddCommand(sendCmd)

	port = sendCmd.Flags().String("port", "9001", "my port")
	bootstrapNode = sendCmd.Flags().StringP("bootstrapNode", "b", "", "bootstrap node id")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sendCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sendCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
