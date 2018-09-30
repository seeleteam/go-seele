/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"errors"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
)

type odrReceipt struct {
	OdrItem
	TxHash    common.Hash
	BlockHash common.Hash
	Index     uint // index in block body
	Receipts  []*types.Receipt
}

var ErrIndexMismatchReceipts = errors.New("error data, index mismatch receipts")
var ErrEmptyBlockHash = errors.New("error data, empty block hash")

func (odr *odrReceipt) code() uint16 {
	return receiptRequestCode
}

func (odr *odrReceipt) handle(lp *LightProtocol) (uint16, odrResponse) {
	txIndex, err := lp.chain.GetStore().GetTxIndex(odr.TxHash)
	if err != nil {
		odr.Error = err.Error()
	}

	receipts, err := lp.chain.GetStore().GetReceiptsByBlockHash(txIndex.BlockHash)
	if err != nil {
		odr.Error = err.Error()
	} else {
		if len(receipts) > 0 {
			odr.Receipts = receipts
			odr.Index = txIndex.Index
			odr.BlockHash = txIndex.BlockHash
		}
	}

	return receiptResponseCode, odr
}

func (odr *odrReceipt) validate(request odrRequest, bcStore store.BlockchainStore) error {
	if odr.Receipts == nil {
		return nil
	}

	if odr.Index < uint(len(odr.Receipts)) {
		return ErrIndexMismatchReceipts
	}

	hash := request.(*odrReceipt).BlockHash
	var header *types.BlockHeader
	var err error
	if hash.IsEmpty() {
		return ErrEmptyBlockHash
	}

	if header, err = bcStore.GetBlockHeader(hash); err != nil {
		return err
	}

	if !hash.Equal(header.ReceiptHash) {
		return types.ErrReceiptRootHash
	}

	return nil
}
