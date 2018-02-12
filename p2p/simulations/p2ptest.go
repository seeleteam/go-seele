package main

import (
	"fmt"
	"net"
	"time"

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
	proto int
}

func (p *myProtocol) Run() {
	fmt.Println("myProtocol Running...", p.proto)
	//	var peer *p2p.Peer
	//	var message *p2p.Message
	ping := time.NewTimer(5 * time.Second)
loop:
	for {
		select {
		case peer := <-p.AddPeerCh:
			fmt.Println("myProtocol new peer", peer)
		case peer := <-p.DelPeerCh:
			fmt.Println("myProtocol del peer", peer)
		case message := <-p.ReadMsgCh:
			fmt.Println("myProtocol readmsg", message)
			break loop
		case <-ping.C:
			//p.SendM
			//fmt.Println("myProtocol ping.C")
			ping.Reset(3 * time.Second)
		}
	}
}

func (p myProtocol) GetBaseProtocol() (baseProto *p2p.Protocol) {
	//fmt.Println("myProtocol Running...")
	return &(p.Protocol)
}

func main() {
	my1 := &myProtocol{
		Protocol: p2p.Protocol{
			Name: "test",

			Version:   1,
			AddPeerCh: make(chan *p2p.Peer),
			DelPeerCh: make(chan *p2p.Peer),
			ReadMsgCh: make(chan *p2p.Message),
		},
	}

	var intFaceL []p2p.ProtocolInterface

	intFaceL = append(intFaceL, my1)
	//	fmt.Println(my1.proto)

	node29 := "0x12345678901234567890123456789012345678901234567890123456789012341234567890123456789012345678901234567890123456789012345678901201"
	node01 := "0x12345678901234567890123456789012345678901234567890123456789012341234567890123456789012345678901234567890123456789012345678901200"

	myType := 1
	//var myServer p2p.Server
	if myType == 1 {
		//slice29, _ := hexutil.HexToBytes(node29)
		//fmt.Println(slice29)
		nodeID29 := common.HexToAddress(node29)
		addr29, _ := net.ResolveUDPAddr("udp4", "182.87.223.29:39009")
		nodeObj29 := discovery.NewNodeWithAddr(nodeID29, addr29)

		myServer := &p2p.Server{
			Config: p2p.Config{
				Name:       "test11",
				ListenAddr: "0.0.0.0:39009",
				KadPort:    "39009",
				MyNodeID:   node01,
			},
		}

		myServer.StaticNodes = append(myServer.StaticNodes, nodeObj29)
		myServer.Protocols = append(myServer.Protocols, my1)
		myServer.Start()
	} else {
		myServer := &p2p.Server{
			Config: p2p.Config{
				Name:       "test29",
				ListenAddr: "0.0.0.0:39009",
				KadPort:    "39009",
				MyNodeID:   node29,
			},
		}
		myServer.Protocols = append(myServer.Protocols, my1)
		myServer.Start()
	}

	time.Sleep(600 * time.Second)
}
