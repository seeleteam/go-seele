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

type odrTriePoof struct {
	odrItem
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

	for k, v := range proof {
		req.Proof = append(req.Proof, proofNode{k, v})
	}

	return trieResponseCode, req
}

func (req *odrTriePoof) handleResponse(resp interface{}) {
	data, ok := resp.(*odrTriePoof)
	if !ok {
		return
	}

	req.Proof = data.Proof
	req.Error = data.Error

	if len(req.Error) > 0 {
		return
	}

	proof := make(map[string][]byte)
	for _, n := range req.Proof {
		proof[n.Key] = n.Value
	}

	if _, err := trie.VerifyProof(req.Root, req.Key, proof); err != nil {
		req.Error = err.Error()
	}
}
