/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import "github.com/seeleteam/go-seele/core/types"

type odrAddTx struct {
	odrItem
	Tx    types.Transaction
	Error string
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
	if data, ok := resp.(odrAddTx); ok {
		req.Error = data.Error
	}
}
