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

var seeleNodeConfigFile *string

// SeeleNodeConfig is test seele node config
type SeeleNodeConfig struct {
	Name      string
	Version   string
	RPCAddr   string
	P2PConfig p2p.Config
}

func seeleNodeConfig(configFile string) (*node.Config, error) {
	seeleNodeConfig := new(SeeleNodeConfig)
	_, err := toml.DecodeFile(configFile, seeleNodeConfig)
	if err != nil {
		return nil, err
	}

	seeleNodeKey, err := crypto.LoadECDSAFromString(seeleNodeConfig.P2PConfig.ECDSAKey)
	if err != nil {
		return nil, err
	}

	return &node.Config{
		Name:    seeleNodeConfig.Name,
		Version: seeleNodeConfig.Version,
		RPCAddr: seeleNodeConfig.RPCAddr,
		P2P: p2p.Config{
			PrivateKey: seeleNodeKey,
			ECDSAKey:   seeleNodeConfig.P2PConfig.ECDSAKey,
		},
	}, nil
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
		seeleService, err := seele.NewSeeleService(0, slog)
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
