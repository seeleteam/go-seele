/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"testing"

	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

func getNode() *discovery.Node {
	return discovery.NewNode(*crypto.MustGenerateRandomAddress(), nil, 0, 1)
}

func Test_NodeSet(t *testing.T) {
	set := NewNodeSet()

	p1 := getNode()
	set.add(p1, false)

	p2 := set.randSelect()
	if p2 == nil {
		t.Fatalf("should select one node.")
	}

	set.delete(p2)
	if set.randSelect() != nil {
		t.Fatalf("should select no node.")
	}
}
