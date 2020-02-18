package cmd

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/log/comm"
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
	"github.com/spf13/cobra"
)

var (
	seeleSubchainNodeConfigFile string
	accountConfig               string
	// voteState                   bool
	subpprofPort      uint64
	subMaxConns       = int(0)
	subMaxActiveConns = int(0)
	vote              string
)

var substartCmd = &cobra.Command{
	Use:   "substart",
	Short: "start the subchain node of seele",
	Long: `useage example:
		   node.exe substart -s subchain_node.json [flags]`,
	Run: func(cmd *cobra.Command, args []string) {
		var wg sync.WaitGroup
		// 1. load config file
		subCfg, err := LoadConfigFromFile(seeleSubchainNodeConfigFile, accountConfig)
		if err != nil {
			fmt.Printf("failed to loading the subchain config file:%+v\n", err.Error())
			return
		}
		if !comm.LogConfiguration.PrintLog {
			fmt.Printf("subchain log folder: %s\n", filepath.Join(log.LogFolder, comm.LogConfiguration.DataDir))
		}
		fmt.Printf("subchain data folder: %s\n", subCfg.BasicConfig.DataDir)

		// 2. new node
		// New node with config/services/log + checkConfig setup(mainly shard)
		// In New, will check the config file first to gurantee some basic config is illegal
		// 2.1
		seeleSubNode, err := node.New(subCfg)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		// 2.2 context
		sclog := log.GetLogger("seeleSubChain")
		subserviceContext := seele.ServiceContext{
			DataDir: subCfg.BasicConfig.DataDir,
		}
		// sctxt := context.WithValue(context.Background(), "SubchainServiceContext", subserviceContext)
		sctxt := context.WithValue(context.Background(), "ServiceContext", subserviceContext)

		// 3. engine
		var engine consensus.Engine
		if subCfg.BasicConfig.MinerAlgorithm == common.BFTSubchainEngine {
			// TODO privateKey can pass with keyfile.
			engine, err = factory.GetBFTSubchainEngine(subCfg.SeeleConfig.CoinbasePrivateKey, subCfg.BasicConfig.DataDir) // SeeleConfig->coinbase privateKey
		} else {
			engine, err = factory.GetConsensusEngine(subCfg.BasicConfig.MinerAlgorithm, subCfg.BasicConfig.DataSetDir)
		}
		if err != nil {
			fmt.Println("failed to get engine with err,", err)
			return
		}

		// debug mode
		if comm.LogConfiguration.IsDebug {
			go monitorPC()
		}

		// 4. http listen and serve
		// start pprof http server
		if subpprofPort > 0 {
			go func() {
				if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", subpprofPort), nil); err != nil {
					fmt.Println("Failed to start subpprof http server,", err)
					return
				}
			}()
		}

		//5. services
		submanager, err := lightclients.NewLightClientManagerSubChain(seeleSubNode.GetShardNumber(), sctxt, subCfg, engine)
		if err != nil {
			fmt.Printf("create light client manager failed. %s\n", err)
			return
		}
		// subservice, err := seele.NewSeeleServiceSubchain(sctxt, subCfg, sclog, engine)
		// when new seeleServices, all iniate works will done inside it
		// 5.1
		subservice, err := seele.NewSeeleService(sctxt, subCfg, sclog, engine, submanager, startHeight)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		// 5.2
		lightServerSubServie, err := light.NewServiceServer(subservice, subCfg, sclog, seeleSubNode.GetShardNumber())
		if err != nil {
			fmt.Println("Create light server err. ", err.Error())
			return
		}
		// 5.3
		monitorSubService, err := monitor.NewMonitorService(subservice, seeleSubNode, subCfg, sclog, "Test SubChain Mointor")
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		// 5.4
		services := submanager.GetServices()
		// services := make([]node.Service, 0)

		// 6. register all services
		services = append(services, subservice, monitorSubService, lightServerSubServie)
		for _, service := range services {
			if err := seeleSubNode.Register(service); err != nil {
				fmt.Println(err.Error())
				return
			}
		}

		// 7. node start: services and RPC
		err = seeleSubNode.Start()

		if subMaxConns > 0 {
			subservice.P2PServer().SetMaxConnections(subMaxConns)
		}
		if subMaxActiveConns > 0 {
			subservice.P2PServer().SetMaxActiveConnections(subMaxActiveConns)
		}
		if err != nil {
			fmt.Printf("got error when start subchain node: %s\n", err)
			return
		}

		// 8. Start miner
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
	substartCmd.Flags().StringVarP(&vote, "vote", "v", "stop", "whether start with voting mode [start/stop], default \"stop\"")
	substartCmd.Flags().Uint64VarP(&subpprofPort, "port", "", 0, "which port pprof http server listen to")
	substartCmd.Flags().IntVarP(&subMaxConns, "maxConns", "", 0, "subchain node max connections")
	substartCmd.Flags().IntVarP(&subMaxActiveConns, "maxActiveConns", "", 0, "subchain node max active connections")
}
