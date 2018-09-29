/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/seeleteam/go-seele/light"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/seeleteam/go-seele/metrics"
	miner2 "github.com/seeleteam/go-seele/miner"
	"github.com/seeleteam/go-seele/monitor"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/seele"
	"github.com/spf13/cobra"
)

var seeleNodeConfigFile string
var miner string
var metricsEnableFlag bool
var accountsConfig string
var threads uint

const (
	lightNode = "light"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the node of seele",
	Long: `usage example:
		node.exe start -c cmd\node.json
		start a node.`,

	Run: func(cmd *cobra.Command, args []string) {
		var wg sync.WaitGroup
		nCfg, err := LoadConfigFromFile(seeleNodeConfigFile, accountsConfig)
		if err != nil {
			fmt.Printf("failed to reading the config file: %s\n", err.Error())
			return
		}

		if !comm.LogConfiguration.PrintLog {
			fmt.Printf("log folder: %s\n", filepath.Join(log.LogFolder, comm.LogConfiguration.DataDir))
		}
		fmt.Printf("data folder: %s\n", nCfg.BasicConfig.DataDir)

		seeleNode, err := node.New(nCfg)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		// Create seele service and register the service
		slog := log.GetLogger("seele")
		lightLog := log.GetLogger("seele-light")
		serviceContext := seele.ServiceContext{
			DataDir: nCfg.BasicConfig.DataDir,
		}
		ctx := context.WithValue(context.Background(), "ServiceContext", serviceContext)

		// default is light node
		if nCfg.BasicConfig.SyncMode == "" {
			nCfg.BasicConfig.SyncMode = lightNode
		}

		if strings.ToLower(nCfg.BasicConfig.SyncMode) == lightNode {
			lightService, err := light.NewServiceClient(ctx, nCfg, lightLog)
			if err != nil {
				fmt.Println("Create light service error.", err.Error())
				return
			}

			if err := seeleNode.Register(lightService); err != nil {
				fmt.Println(err.Error())
				return
			}

			err = seeleNode.Start()
			if err != nil {
				fmt.Printf("got error when start node: %s\n", err)
				return
			}
		} else {
			// fullnode mode
			seeleService, err := seele.NewSeeleService(ctx, nCfg, slog)
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			seeleService.Miner().SetThreads(threads)

			lightServerService, err := light.NewServiceServer(seeleService, nCfg, lightLog)
			if err != nil {
				fmt.Println("Create light server err. ", err.Error())
				return
			}

			// monitor service
			monitorService, err := monitor.NewMonitorService(seeleService, seeleNode, nCfg, slog, "Test monitor")
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			services := []node.Service{seeleService, monitorService, lightServerService}
			for _, service := range services {
				if err := seeleNode.Register(service); err != nil {
					fmt.Println(err.Error())
					return
				}
			}

			err = seeleNode.Start()
			if err != nil {
				fmt.Printf("got error when start node: %s\n", err)
				return
			}

			minerInfo := strings.ToLower(miner)
			if minerInfo == "start" {
				err = seeleService.Miner().Start()
				if err != nil && err != miner2.ErrMinerIsRunning {
					fmt.Println("failed to start the miner : ", err)
					return
				}
			} else if minerInfo == "stop" {
				seeleService.Miner().Stop()
			} else {
				fmt.Println("invalid miner command, must be start or stop")
				return
			}
		}

		if metricsEnableFlag {
			metrics.StartMetricsWithConfig(
				nCfg.MetricsConfig,
				slog,
				nCfg.BasicConfig.Name,
				nCfg.BasicConfig.Version,
				nCfg.P2PConfig.NetworkID,
				nCfg.SeeleConfig.Coinbase,
			)
		}

		wg.Add(1)
		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringVarP(&seeleNodeConfigFile, "config", "c", "", "seele node config file (required)")
	startCmd.MarkFlagRequired("config")

	startCmd.Flags().StringVarP(&miner, "miner", "m", "start", "miner start or not, [start, stop]")
	startCmd.Flags().BoolVarP(&metricsEnableFlag, "metrics", "t", false, "start metrics")
	startCmd.Flags().StringVarP(&accountsConfig, "accounts", "", "", "init accounts info")
	startCmd.Flags().UintVarP(&threads, "threads", "", 1, "miner thread value")
}
