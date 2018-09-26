/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

// ODR object to send tx.
type odrAddTx struct {
	OdrItem
	Tx types.Transaction
}

func (req *odrAddTx) code() uint16 {
	return addTxRequestCode
}

func (req *odrAddTx) handleRequest(lp *LightProtocol) (uint16, odrResponse) {
	if err := lp.txPool.AddTransaction(&req.Tx); err != nil {
		req.Error = err.Error()
	}

	return addTxResponseCode, req
}

func (req *odrAddTx) handleResponse(resp interface{}) odrResponse {
	data, ok := resp.(*odrAddTx)
	if ok {
		req.Error = data.Error
	}

	return data
}

// ODR object to get transaction by hash.
type odrTxByHashRequest struct {
	OdrItem
	TxHash common.Hash
}

type odrTxByHashResponse struct {
	OdrItem
	Tx         *types.Transaction `rlp:"nil"`
	Debt       *types.Debt        `rlp:"nil"`
	BlockIndex *api.BlockIndex    `rlp:"nil"`
	Proof      []proofNode
}

func (req *odrTxByHashRequest) code() uint16 {
	return txByHashRequestCode
}

func (req *odrTxByHashRequest) handleRequest(lp *LightProtocol) (uint16, odrResponse) {
	var err error
	var result odrTxByHashResponse
	result.Tx, result.BlockIndex, result.Debt, err = api.GetTransaction(lp.txPool, lp.chain.GetStore(), req.TxHash)
	result.ReqID = req.ReqID

	if err != nil {
		req.Error = err.Error()
	}

	if result.Tx != nil && result.BlockIndex != nil && result.BlockIndex.BlockHash != common.EmptyHash {
		block, err := lp.chain.GetStore().GetBlock(result.BlockIndex.BlockHash)
		if err != nil {
			req.Error = err.Error()
		}

		txTrie := types.GetTxTrie(block.Transactions)
		proof, err := txTrie.GetProof(req.TxHash.Bytes())
		if err != nil {
			req.Error = err.Error()
		}

		result.Proof = mapToArray(proof)
	}

	return txByHashResponseCode, &result
}

func (req *odrTxByHashRequest) handleResponse(resp interface{}) odrResponse {
	data, ok := resp.(*odrTxByHashResponse)
	if !ok {
		return data
	}

	if len(data.Error) > 0 {
		return data
	}

	if !req.TxHash.Equal(data.Tx.Hash) {
		data.Error = types.ErrHashMismatch.Error()
	}

	if err := data.validate(); err != nil {
		data.Error = err.Error()
	}

	return data
}

func (res *odrTxByHashResponse) validate() error {
	if res.Tx == nil {
		return nil
	}

	if err := res.Tx.ValidateWithoutState(true, false); err != nil {
		return err
	}

	return nil
}
