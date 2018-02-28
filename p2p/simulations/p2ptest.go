package main

import (
	"fmt"
	"net"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

// this package shows how to use p2p network in a higher level
// 1. Create SubProtocol inherits p2p.Protocol
// 2. Implements two functions (Run GetBaseProtocol)
// 3. In function Run, handle three channel
// 4. Append SubProtocols to p2p.server.Protocols

type myProtocol struct {
	p2p.Protocol
	peers map[*p2p.Peer]bool // for test
}

func (p *myProtocol) Run() {
	fmt.Println("myProtocol Running...")
	p.peers = make(map[*p2p.Peer]bool)
	//	var peer *p2p.Peer
	//	var message *p2p.Message
	ping := time.NewTimer(5 * time.Second)

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
			//p.SendM
			fmt.Println("myProtocol ping.C. peers num=", len(p.peers))
			p.sendMyMessage()
			ping.Reset(3 * time.Second)
		}
	}
}

func (p myProtocol) GetBaseProtocol() (baseProto *p2p.Protocol) {
	//fmt.Println("myProtocol Running...")
	return &(p.Protocol)
}

func (p *myProtocol) sendMyMessage() {
	for peer, _ := range p.peers {
		peer.SendMsg(&p.Protocol, 100, []interface{}{"Hello", "world"})
	}
}

type Config struct {
	P2PConfig    p2p.Config
	RemoteNodeID string // optional. peer node
	RemoteAddr   string // optional, format 182.87.223.29:39008
}

func main() {
	var config *Config = new(Config)
	_, err := toml.DecodeFile("test.toml", config)
	if err != nil {
		fmt.Println(err)
		return
	}
	// no config check
	myServer := &p2p.Server{
		Config: config.P2PConfig,
	}

	if config.RemoteNodeID == "" {
		fmt.Println("No remote peer configed, so is a static peer")
	} else {
		nodeIDPeer := common.HexToAddress(config.RemoteNodeID)
		addrPeer, _ := net.ResolveUDPAddr("udp4", config.RemoteAddr)
		nodeObjPeer := discovery.NewNodeWithAddr(nodeIDPeer, addrPeer)
		myServer.StaticNodes = append(myServer.StaticNodes, nodeObjPeer)
	}

	my1 := &myProtocol{
		Protocol: p2p.Protocol{
			Name: "test",

			Version:   1,
			AddPeerCh: make(chan *p2p.Peer),
			DelPeerCh: make(chan *p2p.Peer),
			ReadMsgCh: make(chan *p2p.Message),
		},
	}
	myServer.Protocols = append(myServer.Protocols, my1)
	myServer.Start()
	for {
		time.Sleep(10 * time.Second)
	}
	//time.Sleep(600 * time.Second)
}
