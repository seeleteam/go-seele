/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package monitor

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/seele"
)

func getTmpConfig() *seele.Config {
	acctAddr := crypto.MustGenerateRandomAddress()

	return &seele.Config{
		TxConf:    *core.DefaultTxPoolConfig(),
		NetworkID: 1,
		Coinbase:  *acctAddr,
	}
}

func createTestAPI() *PublicMonitorAPI {
	seeleConf := getTmpConfig()

	testConf := node.Config{
		Name:    "Node for test",
		Version: "Test 1.0",
		DataDir: "node1",
		P2P: p2p.Config{
			ECDSAKey:   "0x6d05f8df3278d8937668f0fd5af7aea6ece8129e615e8586535a021f1626416d",
			ListenAddr: "0.0.0.0:39007",
		},
		RPCAddr:     "127.0.0.1:8080",
		SeeleConfig: *seeleConf,
	}

	serviceContext := seele.ServiceContext{
		DataDir: common.GetTempFolder(),
	}

	ctx := context.WithValue(context.Background(), "ServiceContext", serviceContext)
	dataDir := ctx.Value("ServiceContext").(seele.ServiceContext).DataDir
	defer os.RemoveAll(dataDir)
	log := log.GetLogger("seele", true)

	seeleNode, err := node.New(&testConf)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	seeleService, err := seele.NewSeeleService(ctx, seeleConf, log)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	monitorService, _ := NewMonitorService(seeleService, seeleNode, &testConf, log, "run test")

	seeleNode.Register(monitorService)
	seeleNode.Register(seeleService)

	api := NewPublicMonitorAPI(monitorService)

	err = seeleNode.Start()
	if err != nil {
		return nil
	}

	seeleNode.StartMiner(seeleService)

	return api
}

func Test_PublicMonitorAPI_Allright(t *testing.T) {
	api := createTestAPI()
	if api == nil {
		t.Fatal()
	}
	nodeInfo := NodeInfo{}
	api.NodeInfo(0, &nodeInfo)

	nodeStats := NodeStats{}
	api.NodeStats(0, &nodeStats)
}
