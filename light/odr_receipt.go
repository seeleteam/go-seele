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

type odrReceipt struct {
	OdrItem
	TxHash  common.Hash
	Receipt *types.Receipt `rlp:"nil"`
}

func (odr *odrReceipt) code() uint16 {
	return receiptRequestCode
}

func (odr *odrReceipt) handle(lp *LightProtocol) (uint16, odrResponse) {
	receipt, err := lp.chain.GetStore().GetReceiptByTxHash(odr.TxHash)
	if err != nil {
		odr.Error = err.Error()
	} else {
		odr.Receipt = receipt
	}

	return receiptResponseCode, odr
}

func (odr *odrReceipt) validate(request odrRequest, bcStore store.BlockchainStore) error {
	return nil
}
