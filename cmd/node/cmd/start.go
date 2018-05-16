/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/monitor"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/seele"
	"github.com/spf13/cobra"
)

var seeleNodeConfigFile *string
var miner *string
var genesisConfigFile *string

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the node of seele",
	Long: `usage example:
		node.exe start -c cmd\node.json
		start a node.`,

	Run: func(cmd *cobra.Command, args []string) {
		var wg sync.WaitGroup
		nCfg, err := LoadConfigFromFile(*seeleNodeConfigFile, *genesisConfigFile)
		if err != nil {
			fmt.Printf("reading the config file failed: %s\n", err.Error())
			return
		}

		// print some config infos
		fmt.Printf("log folder: %s\n", log.LogFolder)
		fmt.Printf("data folder: %s\n", nCfg.DataDir)

		seeleNode, err := node.New(nCfg)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		// Create seele service and register the service
		slog := log.GetLogger("seele", common.PrintLog)
		serviceContext := seele.ServiceContext{
			DataDir: nCfg.DataDir,
		}
		ctx := context.WithValue(context.Background(), "ServiceContext", serviceContext)
		seeleService, err := seele.NewSeeleService(ctx, &nCfg.SeeleConfig, slog)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		// monitor service
		monitorService, err := monitor.NewMonitorService(seeleService, seeleNode, nCfg, slog, "Test monitor")
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		services := []node.Service{seeleService, monitorService}
		for _, service := range services {
			if err := seeleNode.Register(service); err != nil {
				fmt.Println(err.Error())
			}
		}

		seeleNode.Start()
		if strings.ToLower(*miner) == "start" {
			err = seeleService.Miner().Start()
			if err != nil {
				fmt.Println("Starting the miner failed: ", err.Error())
				return
			}
		}

		wg.Add(1)
		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	seeleNodeConfigFile = startCmd.Flags().StringP("config", "c", "", "seele node config file (required)")
	startCmd.MarkFlagRequired("config")

	miner = startCmd.Flags().StringP("miner", "m", "start", "miner start or not, [start, stop]")

	genesisConfigFile = startCmd.Flags().StringP("genesis", "g", "", "seele genesis config file")
	startCmd.MarkFlagRequired("genesis")
}
