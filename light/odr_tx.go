/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
)

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
	OdrProvableResponse
	Tx *types.Transaction `rlp:"nil"`
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
		return newErrorResponse(txByHashResponseCode, req.ReqID, err)
	}

	if result.Tx != nil && result.BlockIndex != nil && !result.BlockIndex.BlockHash.IsEmpty() {
		block, err := lp.chain.GetStore().GetBlock(result.BlockIndex.BlockHash)
		if err != nil {
			err = errors.NewStackedErrorf(err, "failed to get block by hash %v", result.BlockIndex.BlockHash)
			return newErrorResponse(txByHashResponseCode, req.ReqID, err)
		}

		txTrie := types.GetTxTrie(block.Transactions)
		proof, err := txTrie.GetProof(req.TxHash.Bytes())
		if err != nil {
			err = errors.NewStackedError(err, "failed to get tx trie proof")
			return newErrorResponse(txByHashResponseCode, req.ReqID, err)
		}

		result.Proof = mapToArray(proof)
	}

	return txByHashResponseCode, &result
}

func (response *odrTxByHashResponse) validateUnpackedTx(txHash common.Hash) error {
	if response.Tx == nil {
		return nil
	}

	if !txHash.Equal(response.Tx.Hash) {
		return types.ErrHashMismatch
	}

	if err := response.Tx.ValidateWithoutState(true, false); err != nil {
		return errors.NewStackedError(err, "failed to validate tx without state")
	}

	return nil
}

func (response *odrTxByHashResponse) validate(request odrRequest, bcStore store.BlockchainStore) error {
	header, err := response.proveHeader(bcStore)
	if err != nil {
		return errors.NewStackedError(err, "failed to prove block header")
	}

	txHash := request.(*odrTxByHashRequest).TxHash

	// tx not packed yet.
	if header == nil {
		return response.validateUnpackedTx(txHash)
	}

	response.Tx = new(types.Transaction)
	if err = response.proveMerkleTrie(header.TxHash, txHash.Bytes(), response.Tx); err != nil {
		return errors.NewStackedError(err, "failed to prove merkle trie")
	}

	return nil
}
