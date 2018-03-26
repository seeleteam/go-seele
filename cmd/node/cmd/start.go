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
	"github.com/seeleteam/go-seele/seele"
	"github.com/spf13/cobra"
)

var seeleNodeConfigFile *string

func seeleNodeConfig(configFile string) (*node.Config, error) {
	seeleNodeConfig := new(node.Config)
	_, err := toml.DecodeFile(configFile, seeleNodeConfig)
	if err != nil {
		return nil, err
	}

	seeleNodeKey, err := crypto.LoadECDSAFromString(seeleNodeConfig.P2P.ECDSAKey)
	if err != nil {
		return nil, err
	}
	seeleNodeConfig.P2P.PrivateKey = seeleNodeKey
	seeleNodeConfig.SeeleConfig.DataRoot = seeleNodeConfig.DataDir

	return seeleNodeConfig, nil
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

		nCfg, err := seeleNodeConfig(*seeleNodeConfigFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		seeleNode, err := node.New(nCfg)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Create seele service and register the service
		slog := log.GetLogger("seele", true)
		seeleService, err := seele.NewSeeleService(&nCfg.SeeleConfig, slog)
		if err != nil {
			fmt.Println(err)
			return
		}
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
