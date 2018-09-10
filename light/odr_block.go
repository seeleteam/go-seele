/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

type odrBlock struct {
	odrItem
	Hash   common.Hash  // Block hash from which to retrieve (excludes Height)
	Height uint64       // Block hash from which to retrieve (excludes Hash)
	Block  *types.Block // Retrieved block
}

func (req *odrBlock) code() uint16 {
	return blockRequestCode
}

func (req *odrBlock) handleRequest(lp *LightProtocol) (uint16, odrResponse) {
	response := *req
	var err error

	if req.Hash.IsEmpty() {
		if response.Block, err = lp.chain.GetStore().GetBlockByHeight(req.Height); err != nil {
			lp.log.Debug("Failed to get block, height = %d, error = %v", req.Height, err)
			return blockResponseCode, &response
		}
	}

	if response.Block, err = lp.chain.GetStore().GetBlock(req.Hash); err != nil {
		lp.log.Debug("Failed to get block, hash = %v, error = %v", req.Hash, err)
		return blockResponseCode, &response
	}

	return blockResponseCode, &response
}

func (req *odrBlock) handleResponse(resp interface{}) {
	if b, ok := resp.(odrBlock); ok {
		req.Block = b.Block
	}
}
