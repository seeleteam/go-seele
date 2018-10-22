/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"bytes"
	"errors"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/trie"
)

type odrReceiptRequest struct {
	OdrItem
	TxHash common.Hash
}

type odrReceiptResponse struct {
	OdrItem
	ReceiptIndex *types.ReceiptIndex `rlp:"nil"`
	Receipt      *types.Receipt      `rlp:"nil"`
	Proof        []proofNode
}

// newOdrReceiptResponse return odrReceiptResponse pointer
func newOdrReceiptResponse() *odrReceiptResponse {
	return &odrReceiptResponse{
		ReceiptIndex: new(types.ReceiptIndex),
		Receipt:      new(types.Receipt),
	}
}

func (odr *odrReceiptRequest) code() uint16 {
	return receiptRequestCode
}

func (odr *odrReceiptRequest) handle(lp *LightProtocol) (uint16, odrResponse) {
	result := newOdrReceiptResponse()
	txIndex, err := lp.chain.GetStore().GetTxIndex(odr.TxHash)
	result.ReqID = odr.ReqID
	if err != nil {
		result.Error = err.Error()
		return receiptResponseCode, result
	}

	receipts, err := lp.chain.GetStore().GetReceiptsByBlockHash(txIndex.BlockHash)
	if err != nil {
		result.Error = err.Error()
	} else if len(receipts) > 0 {
		result.Receipt = receipts[txIndex.Index]
		result.ReceiptIndex.Index = txIndex.Index
		result.ReceiptIndex.BlockHash = txIndex.BlockHash

		receiptTrie := types.GetReceiptTrie(receipts)
		proof, err := receiptTrie.GetProof(crypto.MustHash(result.Receipt).Bytes())
		if err != nil {
			result.Error = err.Error()
			return receiptResponseCode, result
		}
		result.Proof = mapToArray(proof)
	}

	return receiptResponseCode, result
}

func (odr *odrReceiptResponse) getRequestID() uint32 {
	return odr.ReqID
}

func (odr *odrReceiptResponse) getError() error {
	return errors.New(odr.Error)
}

func (odr *odrReceiptResponse) validate(request odrRequest, bcStore store.BlockchainStore) error {
	if odr.Receipt == nil {
		return nil
	}

	txHash := request.(*odrReceiptRequest).TxHash
	if !txHash.Equal(odr.Receipt.TxHash) {
		return types.ErrHashMismatch
	}

	// validate the receipt trie proof if stored in blockchain already.
	if odr.ReceiptIndex != nil {
		header, err := bcStore.GetBlockHeader(odr.ReceiptIndex.BlockHash)
		if err != nil {
			return err
		}

		proof := arrayToMap(odr.Proof)
		value, err := trie.VerifyProof(header.ReceiptHash, crypto.MustHash(odr.Receipt).Bytes(), proof)
		if err != nil {
			return err
		}

		buff := common.SerializePanic(odr.Receipt)
		if !bytes.Equal(buff, value) {
			return errReceiptVerifyFailed
		}
	} else {
		return errReceipIndexNil
	}

	return nil
}
