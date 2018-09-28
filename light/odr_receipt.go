/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

type odrtReceipt struct {
	OdrItem
	Receipt *types.Receipt
	TxHash  common.Hash
}

func (req *odrtReceipt) code() uint16 {
	return receiptRequestCode
}

func (req *odrtReceipt) handleRequest(lp *LightProtocol) (uint16, odrResponse) {
	receipt, err := lp.chain.GetStore().GetReceiptByTxHash(req.TxHash)
	if err != nil {
		req.Error = err.Error()
	} else {
		req.Receipt = receipt
	}

	return receiptResponseCode, req
}

func (req *odrtReceipt) handleResponse(resp interface{}) odrResponse {
	data, ok := resp.(*odrtReceipt)
	if ok {
		req.Error = data.Error
		req.Receipt = data.Receipt
	}

	return data
}
