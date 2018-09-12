/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
)

type odrTrie struct {
	odrItem
	Root  common.Hash
	Key   []byte
	Proof map[string][]byte
}

func (req *odrTrie) code() uint16 {
	return trieRequestCode
}

func (req *odrTrie) handleRequest(lp *LightProtocol) (uint16, odrResponse) {
	statedb, err := lp.chain.GetState(req.Root)
	if err != nil {
		req.Error = err.Error()
		return trieResponseCode, req
	}

	if req.Proof, err = statedb.Trie().GetProof(req.Key); err != nil {
		req.Error = err.Error()
		return trieResponseCode, req
	}

	return trieResponseCode, req
}

func (req *odrTrie) handleResponse(resp interface{}) {
	if data, ok := resp.(*odrTrie); ok {
		req.Proof = data.Proof
		req.Error = data.Error
	}
}
