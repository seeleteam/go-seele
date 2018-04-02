/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/spf13/cobra"
)

var (
	configFile *string
)

// this file shows how to use p2p network in a higher level
// 1. Create SubProtocol inherits p2p.Protocol
// 2. Implements two functions (Run GetBaseProtocol)
// 3. In function Run, handle three channels
// 4. Append SubProtocols to p2p.server.Protocols
// 5. Send hello world message to peers

type ProtocolTest struct {
	p2p.Protocol
	peers []*p2p.Peer

	wg sync.WaitGroup
}

func NewProtocolTest() *ProtocolTest {
	test := &ProtocolTest{
		peers: make([]*p2p.Peer, 0),
		wg:    sync.WaitGroup{},
	}

	test.Protocol = p2p.Protocol{
		Name:    "test",
		Version: 1,
		Length:  1,
		AddPeer: func(peer *p2p.Peer, rw p2p.MsgReadWriter) {
			test.peers = append(test.peers, peer)

			test.wg.Add(2)
			go test.writeMsg(rw)
			go test.handleMsg(rw)

			test.wg.Wait()

			fmt.Println("test done")
		},
		DeletePeer: func(peer *p2p.Peer) {
			// do nothing
		},
	}

	return test
}

func (t *ProtocolTest) writeMsg(rw p2p.MsgWriter) {
	defer t.wg.Done()

	fmt.Println("myProtocol ping.C. peers num=", len(t.peers))
	strs := []string{"Hello", "world"}
	msg := p2p.Message{
		Code:    0,
		Payload: common.SerializePanic(strs),
	}

	rw.WriteMsg(msg)
}

func (t *ProtocolTest) handleMsg(rw p2p.MsgReadWriter) {
	defer t.wg.Done()

	msg, err := rw.ReadMsg()
	if err != nil {
		panic(err)
	}

	str := []string{}
	err = common.Deserialize(msg.Payload, &str)
	if err != nil {
		panic(err)
	}

	fmt.Println(str)
}

func startServer(configFile string) {
	config, err := GetConfigFromFile(configFile)
	if err != nil {
		fmt.Printf("read config file failed %s", err.Error())
		return
	}

	p2pconfig, err := GetP2pConfig(config)
	if err != nil {
		fmt.Printf("generate p2p config failed %s", err.Error())
		return
	}

	myServer := &p2p.Server{
		Config: p2pconfig,
	}

	if len(myServer.StaticNodes) == 0 {
		fmt.Println("No remote peer configed, so is a static peer")
	}

	test := NewProtocolTest()
	myServer.Protocols = append(myServer.Protocols, test.Protocol)
	err = myServer.Start()
	if err != nil {
		fmt.Println("Start err.", err)
		os.Exit(1)
	}

	myServer.Wait()
}

// testCmd represents the test p2p protocol command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "start the p2p server of seele",
	Long: `usage example:
		node.exe test -c cmd\node1.json
		start a p2p server with config file.`,

	Run: func(cmd *cobra.Command, args []string) {
		startServer(*configFile)
	},
}

func init() {
	rootCmd.AddCommand(testCmd)

	configFile = testCmd.Flags().StringP("config", "c", "", "node config (required)")
	testCmd.MarkFlagRequired("config")
}
