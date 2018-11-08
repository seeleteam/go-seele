/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/trie"
)

type proofNode struct {
	Key   string
	Value []byte
}

func arrayToMap(nodes []proofNode) map[string][]byte {
	proof := make(map[string][]byte)
	for _, n := range nodes {
		proof[n.Key] = n.Value
	}

	return proof
}

func mapToArray(proof map[string][]byte) []proofNode {
	var nodes []proofNode
	for k, v := range proof {
		nodes = append(nodes, proofNode{k, v})
	}

	return nodes
}

type odrTriePoof struct {
	OdrItem
	Root  common.Hash
	Key   []byte
	Proof []proofNode
}

func (odr *odrTriePoof) code() uint16 {
	return trieRequestCode
}

func (odr *odrTriePoof) handle(lp *LightProtocol) (uint16, odrResponse) {
	statedb, err := lp.chain.GetState(odr.Root)
	if err != nil {
		odr.Error = errors.NewStackedErrorf(err, "failed to get statedb by root hash %v", odr.Root).Error()
		return trieResponseCode, odr
	}

	proof, err := statedb.Trie().GetProof(odr.Key)
	if err != nil {
		odr.Error = errors.NewStackedError(err, "failed to get trie proof").Error()
		return trieResponseCode, odr
	}

	odr.Proof = mapToArray(proof)
	return trieResponseCode, odr
}

func (odr *odrTriePoof) validate(request odrRequest, bcStore store.BlockchainStore) error {
	proofRequest := request.(*odrTriePoof)
	proof := arrayToMap(odr.Proof)

	if _, err := trie.VerifyProof(proofRequest.Root, proofRequest.Key, proof); err != nil {
		return errors.NewStackedError(err, "failed to verify the trie proof")
	}

	return nil
}
