/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"bytes"

	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/trie"
)

var errForkTx = errors.New("get a transaction from a fork chain")

// ODR object to send tx.
type odrAddTx struct {
	OdrItem
	Tx types.Transaction
}

func (odr *odrAddTx) code() uint16 {
	return addTxRequestCode
}

func (odr *odrAddTx) handle(lp *LightProtocol) (uint16, odrResponse) {
	if err := lp.txPool.AddTransaction(&odr.Tx); err != nil {
		odr.Error = errors.NewStackedError(err, "failed to add tx").Error()
	}

	return addTxResponseCode, odr
}

func (odr *odrAddTx) validate(request odrRequest, bcStore store.BlockchainStore) error {
	return nil
}

// ODR object to get transaction by hash.
type odrTxByHashRequest struct {
	OdrItem
	TxHash common.Hash
}

type odrTxByHashResponse struct {
	OdrItem
	Tx         *types.Transaction `rlp:"nil"`
	BlockIndex *api.BlockIndex    `rlp:"nil"`
	Proof      []proofNode
}

func newOrdTxByHashErrorResponse(reqID uint32, err error) *odrTxByHashResponse {
	return &odrTxByHashResponse{
		OdrItem: OdrItem{
			ReqID: reqID,
			Error: err.Error(),
		},
	}
}

func (req *odrTxByHashRequest) code() uint16 {
	return txByHashRequestCode
}

func (req *odrTxByHashRequest) handle(lp *LightProtocol) (uint16, odrResponse) {
	var err error
	var result odrTxByHashResponse
	result.Tx, result.BlockIndex, err = api.GetTransaction(lp.txPool, lp.chain.GetStore(), req.TxHash)
	result.ReqID = req.ReqID

	if err != nil {
		err = errors.NewStackedErrorf(err, "failed to get tx by hash %v", req.TxHash)
		return txByHashResponseCode, newOrdTxByHashErrorResponse(req.ReqID, err)
	}

	if result.Tx != nil && result.BlockIndex != nil && !result.BlockIndex.BlockHash.IsEmpty() {
		block, err := lp.chain.GetStore().GetBlock(result.BlockIndex.BlockHash)
		if err != nil {
			err = errors.NewStackedErrorf(err, "failed to get block by hash %v", result.BlockIndex.BlockHash)
			return txByHashResponseCode, newOrdTxByHashErrorResponse(req.ReqID, err)
		}

		txTrie := types.GetTxTrie(block.Transactions)
		proof, err := txTrie.GetProof(req.TxHash.Bytes())
		if err != nil {
			err = errors.NewStackedError(err, "failed to get tx trie proof")
			return txByHashResponseCode, newOrdTxByHashErrorResponse(req.ReqID, err)
		}

		result.Proof = mapToArray(proof)
	}

	return txByHashResponseCode, &result
}

func (response *odrTxByHashResponse) validate(request odrRequest, bcStore store.BlockchainStore) error {
	if response.Tx == nil {
		return nil
	}

	txHash := request.(*odrTxByHashRequest).TxHash
	if !txHash.Equal(response.Tx.Hash) {
		return types.ErrHashMismatch
	}

	if err := response.Tx.ValidateWithoutState(true, false); err != nil {
		return errors.NewStackedError(err, "failed to validate tx without state")
	}

	// validate the tx trie proof if stored in blockchain already.
	if response.BlockIndex != nil {
		header, err := bcStore.GetBlockHeader(response.BlockIndex.BlockHash)
		if err != nil {
			return errors.NewStackedErrorf(err, "failed to get block header by hash %v", response.BlockIndex.BlockHash)
		}

		blockHash, err := bcStore.GetBlockHash(response.BlockIndex.BlockHeight)
		if err != nil {
			return errors.NewStackedErrorf(err, "failed to get block hash by height %d", header.Height)
		}
		if !blockHash.Equal(header.Hash()) {
			return errForkTx
		}

		proof := arrayToMap(response.Proof)
		value, err := trie.VerifyProof(header.TxHash, txHash.Bytes(), proof)
		if err != nil {
			return errors.NewStackedError(err, "failed to verify the tx trie proof")
		}

		buff := common.SerializePanic(response.Tx)
		if !bytes.Equal(buff, value) {
			return errTransactionVerifyFailed
		}
	}

	return nil
}
