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
	Height int64        // Block hash from which to retrieve (excludes Hash)
	Block  *types.Block // Retrieved block
}

func (req *odrBlock) code() uint16 {
	return blockRequestCode
}

func (req *odrBlock) handleRequest(lp *LightProtocol) (uint16, odrResponse) {
	var err error
	var h uint64
	if req.Height <= 0 {
		h = lp.chain.CurrentHeader().Height
	} else {
		h = uint64(req.Height)
	}

	if req.Hash.IsEmpty() {
		if req.Block, err = lp.chain.GetStore().GetBlockByHeight(h); err != nil {
			lp.log.Debug("Failed to get block, height = %d, error = %v", req.Height, err)
			req.Error = err.Error()
		}
	} else {
		if req.Block, err = lp.chain.GetStore().GetBlock(req.Hash); err != nil {
			lp.log.Debug("Failed to get block, hash = %v, error = %v", req.Hash, err)
			req.Error = err.Error()
		}
	}

	return blockResponseCode, req
}

func (req *odrBlock) handleResponse(resp interface{}) {
	if data, ok := resp.(*odrBlock); ok {
		req.Error = data.Error
		req.Block = data.Block
	}
}

// Validate validates the retrieved block.
func (req *odrBlock) Validate(bcStore store.BlockchainStore) error {
	if req.Block == nil {
		return nil
	}

	var err error
	if err = req.Block.Validate(); err != nil {
		return err
	}

	hash := req.Hash
	if hash.IsEmpty() {
		if hash, err = bcStore.GetBlockHash(uint64(req.Height)); err != nil {
			return err
		}
	}

	if !hash.Equal(req.Block.HeaderHash) {
		return types.ErrBlockHashMismatch
	}

	return nil
}
