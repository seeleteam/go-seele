/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/seele"
	"github.com/spf13/cobra"
)

var (
	seeleNodeKey, _     = crypto.GenerateKey()
	seeleNodeConfigFile *string
)

// SeeleNodeConfig is test seele node config
type SeeleNodeConfig struct {
	Name          string
	Version       string
	RPCAddr       string
	SeeleECDSAKey string
}

func seeleNodeConfig(configFile string) *node.Config {
	seeleNodeConfig := new(SeeleNodeConfig)
	_, err := toml.DecodeFile(configFile, seeleNodeConfig)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return &node.Config{
		Name:    seeleNodeConfig.Name,
		Version: seeleNodeConfig.Version,
		RPCAddr: seeleNodeConfig.RPCAddr,
		P2P: p2p.Config{
			PrivateKey: seeleNodeKey,
			ECDSAKey:   seeleNodeConfig.SeeleECDSAKey,
		},
	}
}

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the node of seele",
	Long: `usage example:
		node.exe start -c cmd\node.toml
		start a node.`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("start called")
		var wg sync.WaitGroup
		// seeleNode := &node.Node{}
		seeleNode, err := node.New(seeleNodeConfig(*seeleNodeConfigFile))
		if err != nil {
			fmt.Println(err)
			return
		}
		slog := log.GetLogger("seele", true)

		seeleService, _ := seele.NewSeeleService(0, slog)
		services := []node.Service{seeleService}
		for _, service := range services {
			if err := seeleNode.Register(service); err != nil {
				fmt.Println(err)
			}
		}

		seeleNode.Start()
		wg.Add(1)
		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	seeleNodeConfigFile = startCmd.Flags().StringP("config", "c", "", "seele node config file (required)")
	startCmd.MarkFlagRequired("config")
}
