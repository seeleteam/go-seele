package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
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
	StaticNodes  []string
	RemoteNodeID string // optional. peer node
	RemoteAddr   string // optional, format 182.87.223.29:39008
}

func myresolve(id string) (*discovery.Node, error) {
	nodeHeader := "snode://"
	id = id[len(nodeHeader):]

	idSplit := strings.Split(id, "@")
	if len(idSplit) != 2 {
		return nil, errors.New("invalidNodeError")
	}

	address, err := hex.DecodeString(idSplit[0])
	if err != nil {
		return nil, err
	}

	publicKey, err := common.NewAddress(address)
	if err != nil {
		return nil, err
	}

	addr, err := net.ResolveUDPAddr("udp", idSplit[1])
	if err != nil {
		return nil, err
	}

	node := discovery.NewNodeWithAddr(publicKey, addr)
	return node, nil
}

func main() {
	var config *Config = new(Config)
	_, err := toml.DecodeFile("test.toml", config)
	if err != nil {
		fmt.Println(err)
		return
	}
	//fmt.Println(config.StaticNodes[0])
	////os.Exit(0)
	// no config check
	myServer := &p2p.Server{
		Config: config.P2PConfig,
	}

	if len(config.StaticNodes) == 0 {
		fmt.Println("No remote peer configed, so is a static peer")
	} else {
		for _, id := range config.StaticNodes {

			n, err := discovery.NewNodeFromString(id)
			//n, err := myresolve(id)
			if err != nil {
				fmt.Println(err)
				return
			}
			myServer.StaticNodes = append(myServer.StaticNodes, n)
		}
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
	err = myServer.Start()
	if err != nil {
		fmt.Println("Start err.", err)
		os.Exit(1)
	}
	for {
		time.Sleep(10 * time.Second)
	}
}

/*
ecdsa private-public key pairs used for test.
The shorter one is the private key, and the other is public key.
The lengt of public key is 65, has fix prefix '04' which can remove in some case.

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
