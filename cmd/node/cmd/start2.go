package cmd

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/seeleteam/go-seele/metrics"
	miner2 "github.com/seeleteam/go-seele/miner"
	"github.com/seeleteam/go-seele/monitor"
	"github.com/seeleteam/go-seele/seele/lightclients"

	"github.com/seeleteam/go-seele/light"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/factory"

	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/seele"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/spf13/cobra"
)

var (
	seeleSubchainNodeConfigFile string
	accountConfig               string
	voteNode                    bool
	subpprofPort                uint64
	subMaxConns                 = int(0)
	subMaxActiveConns           = int(0)
	vote                        string
)

var substartCmd = &cobra.Command{
	Use:   "substart",
	Short: "start the subchain node of seele",
	Long: `useage example:
		   node.exe substart -s subchain_node.json [flags]`,
	Run: func(cmd *cobra.Command, args []string) {
		var wg sync.WaitGroup
		subCfg, err := LoadConfigFromFile(seeleSubchainNodeConfigFile, accountConfig)
		if err != nil {
			fmt.Printf("failed to loading the subchain config file:%+v\n", err.Error())
			return
		}
		if !comm.LogConfiguration.PrintLog {
			fmt.Printf("subchain log folder: %s\n", filepath.Join(log.LogFolder, comm.LogConfiguration.DataDir))
		}
		fmt.Printf("subchain data folder: %s\n", subCfg.BasicConfig.DataDir)

		// New node with config/services/log + checkConfig setup(mainly shard)
		seeleSubNode, err := node.New(subCfg, true) //
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		// start engine
		sclog := log.GetLogger("seeleSubChain")
		subserviceContext := seele.ServiceContext{
			DataDir: subCfg.BasicConfig.DataDir,
		}
		sctxt := context.WithValue(context.Background(), "SubChainServiceContext", subserviceContext)
		var engine consensus.Engine
		if subCfg.BasicConfig.MinerAlgorithm == common.BFTSubchainEngine {
			engine, err = factory.GetBFTSubchainEngine(subCfg.SeeleConfig.CoinbasePrivateKey, subCfg.BasicConfig.DataDir) // SeeleConfig?
		} else {
			engine, err = factory.GetConsensusEngine(subCfg.BasicConfig.MinerAlgorithm, subCfg.BasicConfig.DataSetDir)
		}
		if err != nil {
			fmt.Println(err)
			return
		}

		// debug mode
		if comm.LogConfiguration.IsDebug {
			go monitorPC()
		}

		// start pprof http server
		if subpprofPort > 0 {
			go func() {
				if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", subpprofPort), nil); err != nil {
					fmt.Println("Failed to start subpprof http server,", err)
					return
				}
			}()
		}

		//services
		manager, err := lightclients.NewLightClientManager(seeleSubNode.GetShardNumber(), sctxt, subCfg, engine)
		if err != nil {
			fmt.Printf("create light client manager failed. %s", err)
			return
		}
		subservice, err := seele.NewSeeleSubService(sctxt, subCfg, sclog, engine)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		lightServerSubServie, err := light.NewServiceServer(subservice, subCfg, nil, seeleSubNode.GetShardNumber())
		if err != nil {
			fmt.Println("Create light server err. ", err.Error())
			return
		}

		monitorSubService, err := monitor.NewMonitorService(subservice, seeleSubNode, subCfg, sclog, "Test SubChain Mointor")
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		services := manager.GetServices()
		services = append(services, subservice, monitorSubService, lightServerSubServie)
		for _, service := range services {
			if err := seeleSubNode.Register(service); err != nil {
				fmt.Println(err.Error())
				return
			}
		}

		// node start
		err = seeleSubNode.Start()
		if subMaxConns > 0 {
			subservice.P2PServer().SetMaxConnections(subMaxConns)
		}
		if subMaxActiveConns > 0 {
			subservice.P2PServer().SetMaxActiveConnections(subMaxActiveConns)
		}
		if err != nil {
			fmt.Printf("got error when start node: %s\n", err)
			return
		}

		// vote status
		voteState := strings.ToLower(vote)
		if voteState == "start" {
			err = subservice.Miner().Start()
			if err != nil && err != miner2.ErrMinerIsRunning {
				fmt.Println("failed to start the miner : ", err)
				return
			}
		} else if voteState == "stop" {
			subservice.Miner().Stop()
		} else {
			fmt.Println("invalid miner command, must be start or stop")
			return
		}

		if metricsEnableFlag {
			metrics.StartMetricsWithConfig(
				subCfg.MetricsConfig,
				sclog,
				subCfg.BasicConfig.Name,
				subCfg.BasicConfig.Version,
				subCfg.P2PConfig.NetworkID,
				subCfg.SeeleConfig.Coinbase,
			)
		}

		wg.Add(1)
		wg.Wait()

	},
}

func init() {
	rootCmd.AddCommand(substartCmd)
	// func (f *FlagSet) StringVarP(p *string, name, shorthand string, value string, usage string)
	substartCmd.Flags().StringVarP(&seeleSubchainNodeConfigFile, "subconfig", "s", "", "seele subchain config file(required)")
	substartCmd.MustMarkFlagRequired("subconfig")
	substartCmd.Flags().StringVarP(&accountConfig, "account", "", "", "init account info")
	substartCmd.Flags().BoolVarP(&voteNode, "vote", "v", false, "whether start with voting mode [start/stop], default \"start\"")
	substartCmd.Flags().Uint64VarP(&subpprofPort, "port", "", 0, "which port pprof http server listen to")
	substartCmd.Flags().IntVarP(&subMaxConns, "maxConns", "", 0, "subchain node max connections")
	substartCmd.Flags().IntVarP(&subMaxActiveConns, "maxActiveConns", "", 0, "subchain node max active connections")
}
