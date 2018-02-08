package main

import (
	"fmt"
	"net"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

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
	fmt.Println(my1.proto)

	node29 := "0x12345678901234567890123456789012345678901234567890123456789012341234567890123456789012345678901234567890123456789012345678901201"
	node01 := "0x12345678901234567890123456789012345678901234567890123456789012341234567890123456789012345678901234567890123456789012345678901200"
	slice29, _ := common.HexToBytes(node29)
	fmt.Println(slice29)
	nodeID29, _ := discovery.BytesToID(slice29)
	addr29, _ := net.ResolveUDPAddr("udp4", "182.87.223.29:39009")
	nodeObj29 := discovery.NewNodeWithAddr(nodeID29, addr29)

	myType := 1
	//var myServer p2p.Server
	if myType == 1 {
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
	fmt.Println(nodeObj29)
	//fmt.Println("main stopped...", time.Second, intFaceL[0].GetBaseProtocol().Name)
	//intFaceL[0].Run()
}
