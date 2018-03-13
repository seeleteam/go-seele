/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/spf13/cobra"
)

var (
	testNodeKey, _ = crypto.GenerateKey()
	testECDSAKey   = "0x445e92837140929b190e89818c39223d1d2b9c07388d80e907adf2e3ba187563"
)

func testNodeConfig() *node.Config {
	return &node.Config{
		Name:    "test node",
		Version: "test version",
		P2P:     p2p.Config{PrivateKey: testNodeKey, ECDSAKey: testECDSAKey},
	}
}

// TestServiceA is a test implementation of the Service interface.
type TestServiceA struct{}

func (s TestServiceA) Protocols() []p2p.ProtocolInterface { return nil }
func (s TestServiceA) Start(*p2p.Server) error            { return nil }
func (s TestServiceA) Stop() error                        { return nil }

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the node of seele",
	Long: `usage example:
    	node start 
		start a node.`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("start called")

		seeleNode, err := node.New(testNodeConfig())
		if err != nil {
			fmt.Println(err)
			return
		}
		var testServicA TestServiceA
		services := []node.Service{testServicA}
		for _, service := range services {
			if err := seeleNode.Register(service); err != nil {
				fmt.Println(err)
			}
		}
		seeleNode.Start()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
