/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
)

type odrBlock struct {
	odrItem
	Hash   common.Hash  // Block hash from which to retrieve (excludes Height)
	Height uint64       // Block hash from which to retrieve (excludes Hash)
	Block  *types.Block // Retrieved block
}

func (ob *odrBlock) code() uint16 {
	return blockRequestCode
}

func (ob *odrBlock) handleRequest(lp *LightProtocol) (uint16, odrResponse) {
	var err error

	if ob.Hash.IsEmpty() {
		if ob.Block, err = lp.chain.GetStore().GetBlockByHeight(ob.Height); err != nil {
			lp.log.Debug("Failed to get block, height = %d, error = %v", ob.Height, err)
			ob.Error = err.Error()
		}
	} else {
		if ob.Block, err = lp.chain.GetStore().GetBlock(ob.Hash); err != nil {
			lp.log.Debug("Failed to get block, hash = %v, error = %v", ob.Hash, err)
			ob.Error = err.Error()
		}
	}

	return blockResponseCode, ob
}

func (ob *odrBlock) handleResponse(resp interface{}) {
	if data, ok := resp.(*odrBlock); ok {
		ob.Error = data.Error
		ob.Block = data.Block
	}
}

// Validate validates the retrieved block.
func (ob *odrBlock) Validate(bcStore store.BlockchainStore) error {
	if ob.Block == nil {
		return nil
	}

	var err error
	if err = ob.Block.Validate(); err != nil {
		return err
	}

	hash := ob.Hash
	if hash.IsEmpty() {
		if hash, err = bcStore.GetBlockHash(ob.Height); err != nil {
			return err
		}
	}

	if !hash.Equal(ob.Block.HeaderHash) {
		return types.ErrBlockHashMismatch
	}

	return nil
}
