/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

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
	seeleNodeConfigFile string
	miner               string
	metricsEnableFlag   bool
	accountsConfig      string
	threads             int

	// default is full node
	lightNode bool

	//pprofPort http server port
	pprofPort uint64

	// profileSize is used to limit when need to collect profiles, set 6GB
	profileSize = uint64(1024 * 1024 * 1024 * 6)
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

		var engine consensus.Engine
		if nCfg.BasicConfig.MinerAlgorithm == common.BFTEngine {
			engine, err = factory.GetBFTEngine(nCfg.SeeleConfig.CoinbasePrivateKey, nCfg.BasicConfig.DataDir)
		} else {
			engine, err = factory.GetConsensusEngine(nCfg.BasicConfig.MinerAlgorithm)
		}

		if err != nil {
			fmt.Println(err)
			return
		}

		// start pprof http server
		if pprofPort > 0 {
			go func() {
				if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", pprofPort), nil); err != nil {
					fmt.Println("Failed to start pprof http server,", err)
					return
				}
			}()
		}

		if comm.LogConfiguration.IsDebug {
			go monitorPC()
		}

		if lightNode {
			lightService, err := light.NewServiceClient(ctx, nCfg, lightLog, common.LightChainDir, seeleNode.GetShardNumber(), engine)
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
			// light client manager
			manager, err := lightclients.NewLightClientManager(seeleNode.GetShardNumber(), ctx, nCfg, engine)
			if err != nil {
				fmt.Printf("create light client manager failed. %s", err)
				return
			}

			// fullnode mode
			seeleService, err := seele.NewSeeleService(ctx, nCfg, slog, engine, manager)
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			seeleService.Miner().SetThreads(threads)

			lightServerService, err := light.NewServiceServer(seeleService, nCfg, lightLog, seeleNode.GetShardNumber())
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

			services := manager.GetServices()
			services = append(services, seeleService, monitorService, lightServerService)
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
	startCmd.Flags().IntVarP(&threads, "threads", "", 1, "miner thread value")
	startCmd.Flags().BoolVarP(&lightNode, "light", "l", false, "whether start with light mode")
	startCmd.Flags().Uint64VarP(&pprofPort, "port", "", 0, "which port pprof http server listen to")
}

func monitorPC() {
	var info runtime.MemStats
	heapDir := filepath.Join(common.GetTempFolder(), "heapProfile")
	err := os.MkdirAll(heapDir, os.ModePerm)
	if err != nil {
		fmt.Printf("failed to create folder %s: %s\n", heapDir, err)
		return
	}

	profileDir := filepath.Join(common.GetTempFolder(), "cpuProfile")
	err = os.MkdirAll(profileDir, os.ModePerm)
	if err != nil {
		fmt.Printf("failed to create folder %s: %s\n", profileDir, err)
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ticker.C:
			runtime.ReadMemStats(&info)
			if info.Alloc > profileSize {
				heapFile := filepath.Join(heapDir, fmt.Sprint("heap-", time.Now().Format("2006-01-02-15-04-05")))
				f, err := os.Create(heapFile)
				if err != nil {
					fmt.Println("monitor create heap file err:", err)
					return
				}
				pprof.WriteHeapProfile(f)

				profileFile := filepath.Join(profileDir, fmt.Sprint("cpu-", time.Now().Format("2006-01-02-15-04-05")))
				cpuf, err := os.Create(profileFile)
				if err != nil {
					fmt.Println("monitor create cpu file err:", err)
					return
				}

				if err := pprof.StartCPUProfile(cpuf); err != nil {
					fmt.Println("failed to start cpu profile err:", err)
					return
				}

				time.Sleep(20 * time.Second)
				pprof.StopCPUProfile()
			}
		}
	}
}
