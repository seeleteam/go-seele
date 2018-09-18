/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/common"
)

type odrtReceipt struct {
	odrItem
	receipt types.Receipt
	Hash   common.Hash
}

func (req *odrtReceipt) code() uint16 {
	return receiptRequestCode
}

func (req *odrtReceipt) handleRequest(lp *LightProtocol) (uint16, odrResponse) {
	if receipt, err := lp.chain.GetStore().GetReceiptByTxHash(req.Hash); err != nil {
		req.receipt = *receipt
		req.Error = err.Error()
	}

	return receiptResponseCode, req
}

func (req *odrtReceipt) handleResponse(resp interface{}) {
	if data, ok := resp.(*odrtReceipt); ok {
		req.Error = data.Error
		req.receipt = data.receipt
	}
}
