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
	key, _ := crypto.GenerateKey()
	testConf := node.Config{
		Name:    "Node for test",
		Version: "Test 1.0",
		DataDir: "node1",
		P2P: p2p.Config{
			PrivateKey: key,
			ListenAddr: "0.0.0.0:39007",
		},
		RPCAddr:     "127.0.0.1:55027",
		WSAddr:      "127.0.0.1:8080",
		WSPattern:   "/ws",
		SeeleConfig: *seeleConf,
	}

	serviceContext := seele.ServiceContext{
		DataDir: common.GetTempFolder() + "/n1/",
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

	seeleService.Miner().Start()

	return api
}

func createTestAPIErr(errBranch int) *PublicMonitorAPI {
	seeleConf := getTmpConfig()

	testConf := node.Config{}
	if errBranch == 1 {

		key, _ := crypto.GenerateKey()
		testConf = node.Config{
			Name:    "Node for test2",
			Version: "Test 1.0",
			DataDir: "node1",
			P2P: p2p.Config{
				PrivateKey: key,
				ListenAddr: "0.0.0.0:39008",
			},
			RPCAddr:     "127.0.0.1:55028",
			SeeleConfig: *seeleConf,
		}
	} else {
		key, _ := crypto.GenerateKey()
		testConf = node.Config{
			Name:    "Node for test3",
			Version: "Test 1.0",
			DataDir: "node1",
			P2P: p2p.Config{
				PrivateKey: key,
				ListenAddr: "0.0.0.0:39009",
			},
			RPCAddr:     "127.0.0.1:55029",
			SeeleConfig: *seeleConf,
		}
	}

	serviceContext := seele.ServiceContext{
		DataDir: common.GetTempFolder() + "/n2/",
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

	if errBranch != 1 {
		seeleNode.Start()
	} else {
		seeleService.Miner().Start()
	}

	return api
}

func Test_PublicMonitorAPI_Allright(t *testing.T) {
	api := createTestAPI()
	if api == nil {
		t.Fatal()
	}
	nodeInfo := NodeInfo{}
	if err := api.NodeInfo(0, &nodeInfo); err != nil {
		t.Fatalf("get nodeInfo failed: %v", err)
	}

	nodeStats := NodeStats{}
	if err := api.NodeStats(0, &nodeStats); err != nil {
		t.Fatalf("get nodeStats failed: %v", err)
	}
}

func Test_PublicMonitorAPI_Err(t *testing.T) {
	api := createTestAPIErr(1)
	if api == nil {
		t.Fatal()
	}
	nodeStats := NodeStats{}
	if err := api.NodeStats(0, &nodeStats); err == nil {
		t.Fatalf("error branch is not covered")
	}
}
