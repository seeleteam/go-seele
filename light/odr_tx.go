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
	odrItem
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

func (req *odrAddTx) handleResponse(resp interface{}) {
	if data, ok := resp.(*odrAddTx); ok {
		req.Error = data.Error
	}
}

// ODR object to get transaction by hash.
type odrTxByHash struct {
	odrItem
	TxHash     common.Hash
	Tx         *types.Transaction
	Debt       *types.Debt
	BlockIndex *api.BlockIndex
}

func (req *odrTxByHash) code() uint16 {
	return addTxRequestCode
}

func (req *odrTxByHash) handleRequest(lp *LightProtocol) (uint16, odrResponse) {
	var err error
	req.Tx, req.BlockIndex, req.Debt, err = api.GetTransaction(lp.txPool, lp.chain.GetStore(), req.TxHash)
	if err != nil {
		req.Error = err.Error()
	}

	return txByHashResponseCode, req
}

func (req *odrTxByHash) handleResponse(resp interface{}) {
	data, ok := resp.(*odrTxByHash)
	if !ok {
		return
	}

	req.Tx = data.Tx
	req.Debt = data.Debt
	req.BlockIndex = data.BlockIndex

	if len(req.Error) > 0 {
		return
	}

	if err := req.validate(); err != nil {
		req.Error = err.Error()
	}
}

func (req *odrTxByHash) validate() error {
	if req.Tx == nil {
		return nil
	}

	if err := req.Tx.ValidateWithoutState(true, false); err != nil {
		return err
	}

	if !req.TxHash.Equal(req.Tx.Hash) {
		return types.ErrHashMismatch
	}

	return nil
}
