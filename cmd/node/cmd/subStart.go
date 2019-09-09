/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/factory"
	"github.com/seeleteam/go-seele/light"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/seeleteam/go-seele/metrics"
	miner2 "github.com/seeleteam/go-seele/miner"
	"github.com/seeleteam/go-seele/monitor"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/seele"
	"github.com/seeleteam/go-seele/seele/lightclients"
	"github.com/spf13/cobra"
)

var (
	seeleSubchainNodeConfigFile string
	subchainMetricsEnableFlag   bool
	subchainAccountsConfig      string
	startSubchainHeight         int

	// default is full node
	subchainLightNode bool

	//subchainpprofPort http server port
	subchainpprofPort uint64

	// subchainProfileSize is used to limit when need to collect profiles, set 6GB
	subchainProfileSize = uint64(1024 * 1024 * 1024 * 6)

	subchainMaxConns       = int(0)
	subchainMaxActiveConns = int(0)
)

// substartCmd represents the subchain start command
var substartCmd = &cobra.Command{
	Use:   "substart",
	Short: "start the subchain node of seele",
	Long: `usage example:
		node.exe substart -sc cmd\node.json
		start a subChain node.`,

	Run: func(cmd *cobra.Command, args []string) {
		var wg sync.WaitGroup
		nCfg, err := LoadConfigFromFile(seeleSubchainNodeConfigFile, subchainAccountsConfig)
		if err != nil {
			fmt.Printf("failed to reading the subchain config file: %s\n", err.Error())
			return
		}

		if !comm.LogConfiguration.PrintLog {
			fmt.Printf("log folder: %s\n", filepath.Join(log.LogFolder, comm.LogConfiguration.DataDir))
		}
		fmt.Printf("data folder: %s\n", nCfg.BasicConfig.DataDir)

		seeleSubchainNode, err := node.New(nCfg)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		// Create seele subchain service and register the services
		slog := log.GetLogger("seele_subchain")
		lightLog := log.GetLogger("seele_subchain-light")
		serviceContext := seele.ServiceContext{
			DataDir: nCfg.BasicConfig.DataDir,
		}
		ctx := context.WithValue(context.Background(), "ServiceContext", serviceContext)

		var engine consensus.Engine
		if nCfg.BasicConfig.MinerAlgorithm == common.BFTEngine {
			engine, err = factory.GetBFTEngine(nCfg.SeeleConfig.CoinbasePrivateKey, nCfg.BasicConfig.DataDir)
		} else {
			engine, err = factory.GetConsensusEngine(nCfg.BasicConfig.MinerAlgorithm, nCfg.BasicConfig.DataSetDir)
		}

		if err != nil {
			fmt.Println(err)
			return
		}

		// start pprof http server
		if subchainpprofPort > 0 {
			go func() {
				if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", subchainpprofPort), nil); err != nil {
					fmt.Println("Failed to start pprof http server,", err)
					return
				}
			}()
		}

		if comm.LogConfiguration.IsDebug {
			go monitorPC()
		}
		// subchian won't have the sharding. Do not need ligthchain
		if subchainLightNode {
			lightService, err := light.NewServiceClient(ctx, nCfg, lightLog, common.LightChainDir, seeleSubchainNode.GetShardNumber(), engine)
			if err != nil {
				fmt.Println("Create light service error.", err.Error())
				return
			}

			if err := seeleSubchainNode.Register(lightService); err != nil {
				fmt.Println(err.Error())
				return
			}

			err = seeleSubchainNode.Start()
			if err != nil {
				fmt.Printf("got error when start node: %s\n", err)
				return
			}
		} else {
			// light client manager
			manager, err := lightclients.NewLightClientManager(seeleSubchainNode.GetShardNumber(), ctx, nCfg, engine)
			if err != nil {
				fmt.Printf("create light client manager failed. %s", err)
				return
			}

			// fullnode mode
			seeleService, err := seele.NewSeeleService(ctx, nCfg, slog, engine, manager, startSubchainHeight)
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			seeleService.Miner().SetThreads(threads)

			lightServerService, err := light.NewServiceServer(seeleService, nCfg, lightLog, seeleSubchainNode.GetShardNumber())
			if err != nil {
				fmt.Println("Create light server err. ", err.Error())
				return
			}

			// monitor service
			monitorService, err := monitor.NewMonitorService(seeleService, seeleSubchainNode, nCfg, slog, "Test monitor")
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			services := manager.GetServices()
			services = append(services, seeleService, monitorService, lightServerService)
			for _, service := range services {
				if err := seeleSubchainNode.Register(service); err != nil {
					fmt.Println(err.Error())
					return
				}
			}

			err = seeleSubchainNode.Start()
			if subchainMaxConns > 0 {
				seeleService.P2PServer().SetMaxConnections(subchainMaxConns)
			}
			if subchainMaxActiveConns > 0 {
				seeleService.P2PServer().SetMaxActiveConnections(subchainMaxActiveConns)
			}
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

		if subchainMetricsEnableFlag {
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
	rootCmd.AddCommand(substartCmd)

	// substartCmd.Flags().StringVarP(&seeleSubchainNodeConfigFile, "subchain_config", "", "", "seele node subchainConfig file (required)")
	// substartCmd.MustMarkFlagRequired("subchain_config")

	substartCmd.Flags().StringVarP(&seeleSubchainNodeConfigFile, "subconfig", "s", "", "seele node config file (required)")
	substartCmd.MustMarkFlagRequired("subconfig")

	substartCmd.Flags().BoolVarP(&subchainMetricsEnableFlag, "subchain_metrics", "d", false, "start metrics")
	substartCmd.Flags().StringVarP(&subchainAccountsConfig, "subchain_accounts", "", "", "init accounts info")
	substartCmd.Flags().BoolVarP(&subchainLightNode, "subchain_light", "", false, "whether start with light mode")
	substartCmd.Flags().Uint64VarP(&subchainpprofPort, "subchain_port", "", 0, "which port pprof http server listen to")
	// substartCmd.Flags().IntVarP(&startSubchainHeight, "subchain_startHeight", "", -1, "the block height to start from")
	substartCmd.Flags().IntVarP(&subchainMaxConns, "subchain_maxConns", "", 0, "node max connections")
	substartCmd.Flags().IntVarP(&subchainMaxActiveConns, "subchain_maxActiveConns", "", 0, "node max active connections")
}
