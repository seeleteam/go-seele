/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	log2 "github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/stretchr/testify/assert"
)

func Test_peer_Info(t *testing.T) {
	// prepare some variables
	myAddr := common.HexMustToAddres("0x6b9fd39a9f1273c46fba8951b62de5b95cd3dd84")
	node1 := discovery.NewNode(myAddr, nil, 0, 0)
	log := log2.GetLogger("test")
	p2pPeer := &p2p.Peer{
		Node: node1,
	}
	var myHash common.Hash
	copy(myHash[0:20], myAddr[:])
	bigInt := big.NewInt(100)
	okStr := "{\"version\":1,\"difficulty\":100,\"head\":\"6b9fd39a9f1273c46fba8951b62de5b95cd3dd84000000000000000000000000\"}"

	// Create peer for test
	peer := newPeer(SeeleVersion, p2pPeer, nil, log)
	peer.SetHead(myHash, bigInt)

	peerInfo := peer.Info()
	data, _ := json.Marshal(peerInfo)
	resultStr := string(data)
	if okStr != resultStr {
		t.Fail()
	}
}

func Test_verifyGenesis(t *testing.T) {
	networkID := uint64(0)
	statusData := statusData{
		ProtocolVersion: uint32(0),
		NetworkID:       networkID,
		TD:              big.NewInt(0),
		CurrentBlock:    common.EmptyHash,
		GenesisBlock:    common.EmptyHash,
		Shard:           1,
		Difficult:       8000000,
	}
	err := verifyGenesisAndNetworkID(statusData, common.EmptyHash, networkID, 1, 8000000)
	assert.Equal(t, err, nil)

	err = verifyGenesisAndNetworkID(statusData, common.EmptyHash, networkID, 2, 8000000)
	assert.Equal(t, err, nil)

	err = verifyGenesisAndNetworkID(statusData, common.EmptyHash, networkID, 2, 9000000)
	assert.Equal(t, err == errGenesisDifficultNotMatch, true)

	errorHash := common.StringToHash("error hash")
	err = verifyGenesisAndNetworkID(statusData, errorHash, networkID, 1, 8000000)
	assert.Equal(t, err != nil, true)
}
