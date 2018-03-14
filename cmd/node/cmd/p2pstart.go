/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/spf13/cobra"
)

var (
	configFile *string
)

// this package shows how to use p2p network in a higher level
// 1. Create SubProtocol inherits p2p.Protocol
// 2. Implements two functions (Run GetBaseProtocol)
// 3. In function Run, handle three channel
// 4. Append SubProtocols to p2p.server.Protocols

type myProtocol struct {
	p2p.Protocol
	peers map[*p2p.Peer]bool //for test
}

func (p *myProtocol) Run() {
	fmt.Println("myProtocol Running...")
	p.peers = make(map[*p2p.Peer]bool)
	ping := time.NewTimer(10 * time.Second)

	for {
		select {
		case peer := <-p.AddPeerCh:
			p.peers[peer] = true
			fmt.Println("myProtocol add new peer. peers=", len(p.peers))
		case peer := <-p.DelPeerCh:
			fmt.Println("myProtocol del peer")
			// need del from peers
			delete(p.peers, peer)
			fmt.Println("myProtocol del peer. peers=", len(p.peers))
		case message := <-p.ReadMsgCh:
			fmt.Println("myProtocol readmsg", message)
		case <-ping.C:
			fmt.Println("myProtocol ping.C. peers num=", len(p.peers))
			p.sendMessage()
			//ping.Reset(3 * time.Second)
		}
	}
}

func (p myProtocol) GetBaseProtocol() (baseProto *p2p.Protocol) {
	//fmt.Println("myProtocol Running...")
	return &(p.Protocol)
}

func (p *myProtocol) sendMessage() {
	for peer := range p.peers {
		peer.SendMsg(&p.Protocol, 100, []interface{}{"Hello", "world"})
	}
}

// Config is test p2p server's config
type Config struct {
	P2PConfig   p2p.Config
	StaticNodes []string
}

func startServer(configFile string) {
	config := new(Config)
	_, err := toml.DecodeFile(configFile, config)
	if err != nil {
		fmt.Println(err)
		return
	}

	myServer := &p2p.Server{
		Config: config.P2PConfig,
	}

	if len(config.StaticNodes) == 0 {
		fmt.Println("No remote peer configed, so is a static peer")
	} else {
		for _, id := range config.StaticNodes {
			n, err := discovery.NewNodeFromString(id)
			if err != nil {
				fmt.Println(err)
				return
			}

			myServer.StaticNodes = append(myServer.StaticNodes, n)
		}
	}

	my1 := &myProtocol{
		Protocol: p2p.Protocol{
			Name:      "test",
			Version:   1,
			AddPeerCh: make(chan *p2p.Peer),
			DelPeerCh: make(chan *p2p.Peer),
			ReadMsgCh: make(chan *p2p.Message),
		},
	}
	myServer.Protocols = append(myServer.Protocols, my1)
	err = myServer.Start()
	if err != nil {
		fmt.Println("Start err.", err)
		os.Exit(1)
	}

	myServer.Wait()
}

// p2pStartCmd represents the start command
var p2pStartCmd = &cobra.Command{
	Use:   "p2pstart",
	Short: "start the p2p server of seele",
	Long: `usage example:
		p2p server start 
		start a p2p server.`,

	Run: func(cmd *cobra.Command, args []string) {
		startServer(*configFile)
	},
}

func init() {
	rootCmd.AddCommand(p2pStartCmd)

	configFile = p2pStartCmd.Flags().StringP("config", "c", "", "node config")
}
