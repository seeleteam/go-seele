/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"crypto/ecdsa"
	"strings"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/stretchr/testify/assert"
)

func Test_NewServer(t *testing.T) {
	var genesis core.GenesisInfo
	config := testConfig()
	server := NewServer(genesis, *config, nil)

	assert.Equal(t, server != nil, true)
	assert.Equal(t, server.Config, *config)
	assert.Equal(t, server.running, false)
	assert.Equal(t, server.MaxPeers, defaultMaxPeers)
	assert.Equal(t, server.MaxPendingPeers, 0)
	assert.Equal(t, server.genesis, genesis)

	// verify the peerSet
	assert.Equal(t, server.peerSet != nil, true)
	assert.Equal(t, server.PeerCount(), 0)
	assert.Equal(t, len(server.peerSet.shardPeerMap), common.ShardCount)
}

func Test_Start(t *testing.T) {
	var genesis core.GenesisInfo
	config := testConfig()
	server := NewServer(genesis, *config, nil)

	// server already started
	server.running = true
	err := server.Start("testDir", 1)
	assert.Equal(t, err != nil, true)
	assert.Equal(t, strings.Contains(err.Error(), "server already running"), true)

	// start server with invalid ListenAddr
	config = testInvalidConfig()
	server = NewServer(genesis, *config, nil)
	err = server.Start("testDir", 1)
	assert.Equal(t, err != nil, true)

	// start server
	config = testConfig()
	server = NewServer(genesis, *config, nil)
	err = server.Start("testDir", 1)
	assert.Equal(t, err, nil)

	// It's ok to stop more than once
	server.Stop()
	server.Stop()
}

func Test_addNode(t *testing.T) {
	var genesis core.GenesisInfo
	config := testConfig()
	server := NewServer(genesis, *config, nil)

	id := *crypto.MustGenerateShardAddress(1)
	node := discovery.MustNewNodeWithAddr(id, "127.0.1.1:9000", 0)
	server.addNode(node)
	assert.Equal(t, server.PeerCount(), 0)

	node = discovery.MustNewNodeWithAddr(id, "127.0.1.1:9000", 1)
	server.addNode(node)
	assert.Equal(t, server.PeerCount(), 0) // failed to connect to this node
}

func Test_deleteNode(t *testing.T) {
	var genesis core.GenesisInfo
	config := testConfig()
	server := NewServer(genesis, *config, nil)
	assert.Equal(t, server.PeerCount(), 0)

	addr := crypto.MustGenerateShardAddress(1).Hex()
	peer1, err := newTestPeer(addr, 1)
	if err != nil {
		t.Fatal(err)
	}
	server.addPeer(peer1)
	assert.Equal(t, server.PeerCount(), 1)

	addr = crypto.MustGenerateShardAddress(1).Hex()
	peer2, err := newTestPeer(addr, 1)
	if err != nil {
		t.Fatal(err)
	}
	server.addPeer(peer2)
	assert.Equal(t, server.PeerCount(), 2)

	server.deleteNode(peer1.Node)
	assert.Equal(t, server.PeerCount(), 1)

	server.deleteNode(peer2.Node)
	assert.Equal(t, server.PeerCount(), 0)
}

func Test_peerIsValidate(t *testing.T) {
	var genesis core.GenesisInfo
	config := testConfig()
	server := NewServer(genesis, *config, testProtocol())
	assert.Equal(t, server.PeerCount(), 0)

	var message = &Message{}
	recvMsg, renounceCnt, err := server.unPackWrapHSMsg(message)
	assert.Equal(t, strings.Contains(err.Error(), "received msg with invalid length"), true)
	assert.Equal(t, renounceCnt, uint64(0))
	assert.Equal(t, recvMsg == nil, true)

	var caps []Cap
	for _, proto := range server.Protocols {
		caps = append(caps, proto.cap())
	}

	handshakeMsg := &ProtoHandShake{Caps: caps}
	handshakeMsg.NetworkID = server.Config.NetworkID
	node := discovery.MustNewNodeWithAddr(*crypto.MustGenerateShardAddress(1), "127.0.1.1:9000", 0)
	message, err = server.packWrapHSMsg(handshakeMsg, node.ID[0:], outboundConn)
	assert.Equal(t, err, nil)

	recvMsg, renounceCnt, err = server.unPackWrapHSMsg(message)
	assert.Equal(t, strings.Contains(err.Error(), " received public key not match"), true)
}

func Test_PeerInfos(t *testing.T) {
	peerInfos := testPeerInfos()

	assert.Equal(t, peerInfos.Len(), 2)
	assert.Equal(t, peerInfos.Less(0, 1), true)
	peerInfos.Swap(0, 1)
	assert.Equal(t, peerInfos.Less(0, 1), false)

	var genesis core.GenesisInfo
	config := testConfig()
	server := NewServer(genesis, *config, testProtocol())

	peerInfoArray := server.PeersInfo()
	assert.Equal(t, len(peerInfoArray), 0)

	addr := crypto.MustGenerateShardAddress(1).Hex()
	peer1, err := newTestPeer(addr, 1)
	if err != nil {
		t.Fatal(err)
	}
	server.addPeer(peer1)

	peerInfoArray = server.PeersInfo()
	assert.Equal(t, len(peerInfoArray), 1)
}

func testConfig() *Config {
	return &Config{
		ListenAddr:    "127.0.0.1:8080",
		NetworkID:     "seele",
		SubPrivateKey: "privKey",
		PrivateKey:    generatePrivKey(),
	}
}

func testInvalidConfig() *Config {
	return &Config{
		ListenAddr:    "127.0.0:8080",
		NetworkID:     "seele",
		SubPrivateKey: "privKey",
		PrivateKey:    generatePrivKey(),
	}
}

func testProtocol() []Protocol {
	return []Protocol{
		{
			Name:    "udp",
			Version: 1,
			Length:  1048,
		},
	}
}

func testPeerInfos() PeerInfos {
	return []PeerInfo{
		{
			ID:    "id1",
			Shard: 1,
		},
		{
			ID:    "id2",
			Shard: 1,
		},
	}
}

func generatePrivKey() *ecdsa.PrivateKey {
	_, keypair, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(err)
	}

	return keypair
}
