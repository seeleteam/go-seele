/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/trie"
)

// OdrProvableResponse represents all provable ODR response.
type OdrProvableResponse struct {
	OdrItem
	BlockIndex *api.BlockIndex `rlp:"nil"`
	Proof      []proofNode
}

// proveHeader proves the response is valid with the specified blockchain store,
// and returns the corresponding block heaer in canonical chain. If the retrieved
// block index is nil, then return nil block header.
func (response *OdrProvableResponse) proveHeader(bcStore store.BlockchainStore) (*types.BlockHeader, error) {
	if response.BlockIndex == nil {
		return nil, nil
	}

	header, err := bcStore.GetBlockHeader(response.BlockIndex.BlockHash)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get block header by hash %v", response.BlockIndex.BlockHash)
	}

	canonicalHash, err := bcStore.GetBlockHash(response.BlockIndex.BlockHeight)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get block hash by height %v", response.BlockIndex.BlockHeight)
	}

	if !canonicalHash.Equal(response.BlockIndex.BlockHash) {
		return nil, types.ErrBlockHashMismatch
	}

	return header, nil
}

// proveMerkleTrie proves the merkle trie in the response with specified root and key.
// If proved, decode the retrieved ODR object to obj (pointer type) from the value
// of leaf node in merkle proof.
func (response *OdrProvableResponse) proveMerkleTrie(root common.Hash, key []byte, obj interface{}) error {
	proof := arrayToMap(response.Proof)

	value, err := trie.VerifyProof(root, key, proof)
	if err != nil {
		return errors.NewStackedError(err, "failed to verify the merkle trie proof")
	}

	if err = common.Deserialize(value, obj); err != nil {
		return errors.NewStackedError(err, "failed to decode the value of leaf node in merkle proof")
	}

	return nil
}
