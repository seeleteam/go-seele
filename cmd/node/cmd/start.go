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

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the node of seele",
	Long: `usage example:
		node.exe start -c cmd\node.json
		start a node.`,

	Run: func(cmd *cobra.Command, args []string) {
		var wg sync.WaitGroup
		nCfg, err := LoadConfigFromFile(*seeleNodeConfigFile)
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
}

/*
ecdsa private-public key pairs used for test.
The shorter one is the private key, and the other is the public key.
The length of public key is 65, with the fixed prefix '04' which can be removed in some cases.
29
key00   692e7dcb0efebc71bd544755baebb41a6b7245efd78799e836d56ef02f417efa
key00   040548d0b1a3297fea072284f86b9fd39a9f1273c46fba8951b62de5b95cd3dd846278057ec4df598a0b089a0bdc0c8fd3aa601cf01a9f30a60292ea0769388d1f
34
key01   c39f90055a5e57302fdc9742441a1e4492c639ce8a157419b36edf1280f9fffe
key01   04fa0b5a1794507ceb4fa67d0228bc038a93c75d6a1a9c6eacb87f20589c03a2409b99add0631961edec47d91451a4e5bf47c28ea1ab28c6cd15d806841db4e6d6
01
key02   445e92837140929b190e89818c39223d1d2b9c07388d80e907adf2e3ba187563
key02   04607a2d7b0d1e899fb3cd3ac2ece65acd888a5de59ab0a215a8533f59c46245f60a6d70766f71738b7b2186b9302f7d1ca1430c082502ec9875ad1d3ea1ff1e29
key03   3f28bb0638f32f86db45d395f87f0ee57f73c18d32cb758ddd0882325e8f5010
key03   0444c2e293ed6cafb9c97c8c2a997f72a39e28527b6d191af5a5fa884cbae44eac0f7ec801e3e70769c103b46b72f17022eaa11e9792ea3eb281de7f55642f7a6f
key04   c9ce34bd13daf376bc288ddc2587e0c3814996fc882221fb135235f7d2e93d0b
key04   04f1b23e9e134c009d6a841a6a5cca5b52e90648ce43213d6c62e0cc2952048c4a36429f0d5fa0c06350c0d747de6a0d5aee8d2eac5c3b5808117e3a9adc4a8e63
key05   6f7b757302d53b508296ea4a0535de38ff026142df3e77eabc4f5a5c719ecaf2
key05   0476745aeb30e3e950ad4b9b2aa4d4336ae083ae96e9ef16999200ee991554f0e44601e5c4c05ee34b84a653f2eac0f300fd128432414b89741fd71b7c9ec59c94
key06   bc9392547036ef418da0036875001ba69f23f14a353084ae4bfee68a9d126a67
key06   04dfb612279d1b6b5462f865a308db38fb9b93312bddc90456e22bcd8cb5c9a7f219c29285ea3ccb9ef939b375b8beb2c479ae0a3a8bc92ce8fec6dfbdd5dd14eb
key07   d5cdc9de17e9d5103a6f62204a41b358f97fcfbf93efd1b821b66ab95cef2556
key07   0427eb6b323b40047191442c90ef5593dd54f2e152a7dae419e491f13cd2c3733b2bf148f7a335768f14224dcd53a775e87e6f1c2f87e19c085bc6c9c4f61e783c
key08   eb0766ca65a1a86bf31991620e79ab09eefd00e968252e6e524ecd998fb7512b
key08   042236c62159d97c83724b24b65a90bf35d354d2c0ad2fabc9bef1e157cf48458f2e54db7daf209c18b77725eeada8337117abbd246b565a4576d21463480169da
key09   49c8d78ce673b1c1f0cf3dd8f037d2187227823d01c77a83e8fe9a83c7a9c867
key09   049e695bc72c0bdf980e56dcc338999787309e2ed995214ab01e024bd87e6ebbea89701b9c441eb6ba0284fcc0f7d1daa27a1b842dc057f5f6bccd5b7675a23dab
*/
