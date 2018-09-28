/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
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

func (req *odrTriePoof) code() uint16 {
	return trieRequestCode
}

func (req *odrTriePoof) handleRequest(lp *LightProtocol) (uint16, odrResponse) {
	statedb, err := lp.chain.GetState(req.Root)
	if err != nil {
		req.Error = err.Error()
		return trieResponseCode, req
	}

	proof, err := statedb.Trie().GetProof(req.Key)
	if err != nil {
		req.Error = err.Error()
		return trieResponseCode, req
	}

	req.Proof = mapToArray(proof)
	return trieResponseCode, req
}

func (req *odrTriePoof) handleResponse(resp interface{}) odrResponse {
	data, ok := resp.(*odrTriePoof)
	if !ok {
		return data
	}

	req.Proof = data.Proof
	req.Error = data.Error

	if len(req.Error) > 0 {
		return data
	}

	proof := arrayToMap(req.Proof)
	if _, err := trie.VerifyProof(req.Root, req.Key, proof); err != nil {
		req.Error = err.Error()
	}

	return data
}
