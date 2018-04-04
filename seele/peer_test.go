/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/p2p/discovery"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/p2p"
)

func Test_peer_Info(t *testing.T) {
	// prepare some variables
	myAddr := common.HexMustToAddres("0x0548d0b1a3297fea072284f86b9fd39a9f1273c46fba8951b62de5b95cd3dd846278057ec4df598a0b089a0bdc0c8fd3aa601cf01a9f30a60292ea0769388d1f")
	node1 := discovery.NewNode(myAddr, nil, 0)
	p2pPeer := &p2p.Peer{
		Node: node1,
	}
	var myHash common.Hash
	copy(myHash[0:], myAddr[0:common.HashLength])
	bigInt := big.NewInt(100)
	okStr := "{\"version\":1,\"difficulty\":100,\"head\":\"0548d0b1a3297fea072284f86b9fd39a9f1273c46fba8951b62de5b95cd3dd84\"}"

	// Create peer for test
	peer := newPeer(SeeleVersion, p2pPeer, nil)
	peer.SetHead(myHash, bigInt)

	peerInfo := peer.Info()
	data, _ := json.Marshal(peerInfo)
	resultStr := string(data)
	if okStr != resultStr {
		t.Fail()
	}
}
