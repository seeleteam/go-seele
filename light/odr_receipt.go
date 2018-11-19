/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"bytes"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/trie"
)

var errForkMessage = errors.New("get message from a fork chain")

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

func (odr *odrReceiptRequest) code() uint16 {
	return receiptRequestCode
}

func (odr *odrReceiptRequest) handle(lp *LightProtocol) (uint16, odrResponse) {
	var result odrReceiptResponse
	result.ReqID = odr.ReqID

	txIndex, err := lp.chain.GetStore().GetTxIndex(odr.TxHash)
	if err != nil {
		result.Error = errors.NewStackedErrorf(err, "failed to get tx index by hash %v", odr.TxHash).Error()
		return receiptResponseCode, &result
	}

	receipts, err := lp.chain.GetStore().GetReceiptsByBlockHash(txIndex.BlockHash)
	if err != nil {
		result.Error = errors.NewStackedErrorf(err, "failed to get receipts by block hash %v", txIndex.BlockHash).Error()
	} else if len(receipts) > 0 {
		result.Receipt = receipts[txIndex.Index]
		result.ReceiptIndex = &types.ReceiptIndex{
			BlockHash: txIndex.BlockHash,
			Index:     txIndex.Index,
		}

		receiptTrie := types.GetReceiptTrie(receipts)
		proof, err := receiptTrie.GetProof(crypto.MustHash(result.Receipt).Bytes())
		if err != nil {
			result.Error = errors.NewStackedError(err, "failed to get receipt trie proof").Error()
			return receiptResponseCode, &result
		}

		result.Proof = mapToArray(proof)
	}

	return receiptResponseCode, &result
}

func (odr *odrReceiptResponse) validate(request odrRequest, bcStore store.BlockchainStore) error {
	if odr.Receipt == nil {
		return nil
	}

	if txHash := request.(*odrReceiptRequest).TxHash; !txHash.Equal(odr.Receipt.TxHash) {
		return types.ErrHashMismatch
	}

	if odr.ReceiptIndex == nil {
		return errReceipIndexNil
	}

	// validate the receipt trie proof if stored in blockchain already.
	header, err := bcStore.GetBlockHeader(odr.ReceiptIndex.BlockHash)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get block header by hash %v", odr.ReceiptIndex.BlockHash)
	}

	blockHash, err := bcStore.GetBlockHash(header.Height)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get block hash by height %d", header.Height)
	}
	if !blockHash.Equal(header.Hash()) {
		return errForkMessage
	}

	proof := arrayToMap(odr.Proof)
	value, err := trie.VerifyProof(header.ReceiptHash, crypto.MustHash(odr.Receipt).Bytes(), proof)
	if err != nil {
		return errors.NewStackedError(err, "failed to verify receipt trie proof")
	}

	if buff := common.SerializePanic(odr.Receipt); !bytes.Equal(buff, value) {
		return errReceiptVerifyFailed
	}

	return nil
}
