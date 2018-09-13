/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
)

type odrTriePoof struct {
	odrItem
	Root  common.Hash
	Key   []byte
	Proof map[string][]byte
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

	if req.Proof, err = statedb.Trie().GetProof(req.Key); err != nil {
		req.Error = err.Error()
	}
	return trieResponseCode, req
}

func (req *odrTriePoof) handleResponse(resp interface{}) {
	if data, ok := resp.(*odrTriePoof); ok {
		req.Proof = data.Proof
		req.Error = data.Error
	}
}

// Get implements the trie.Database interface.
func (req *odrTriePoof) Get(key []byte) ([]byte, error) {
	if req.Proof == nil {
		return nil, nil
	}

	return req.Proof[string(key)], nil
}
