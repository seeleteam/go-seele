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

type odrReceiptRequest struct {
	OdrItem
	TxHash common.Hash
}

type odrReceiptResponse struct {
	OdrItem
	BlockHash common.Hash
	Index     uint // index in block body
	Receipts  []*types.Receipt
}

var (
	ErrIndexMismatchReceipts = errors.New("error data, index mismatch receipts")
	ErrMismatchTxHash        = errors.New("error data, mismatch tx hash")
)

func (odr *odrReceiptRequest) code() uint16 {
	return receiptRequestCode
}

func (odr *odrReceiptRequest) handle(lp *LightProtocol) (uint16, odrResponse) {
	txIndex, err := lp.chain.GetStore().GetTxIndex(odr.TxHash)
	if err != nil {
		odr.Error = err.Error()
	}

	var result odrReceiptResponse
	receipts, err := lp.chain.GetStore().GetReceiptsByBlockHash(txIndex.BlockHash)
	if err != nil {
		result.Error = err.Error()
	} else if len(receipts) > 0 {
		result.Receipts = receipts
		result.Index = txIndex.Index
		result.BlockHash = txIndex.BlockHash
	}

	return receiptResponseCode, &result
}

func (odr *odrReceiptResponse) validate(request odrRequest, bcStore store.BlockchainStore) error {
	if odr.Receipts == nil {
		return nil
	}

	if odr.Index < uint(len(odr.Receipts)) {
		return ErrIndexMismatchReceipts
	}

	var header *types.BlockHeader
	var err error

	if header, err = bcStore.GetBlockHeader(odr.BlockHash); err != nil {
		return err
	}

	rceiptMerkleRootHash := types.ReceiptMerkleRootHash(odr.Receipts)
	if !rceiptMerkleRootHash.Equal(header.ReceiptHash) {
		return types.ErrReceiptRootHash
	}

	txhash := request.(*odrReceiptRequest).TxHash
	if txhash != odr.Receipts[odr.Index].TxHash {
		return ErrMismatchTxHash
	}

	return nil
}
