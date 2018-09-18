/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	api2"github.com/seeleteam/go-seele/api"
)

// TransactionPoolAPI provides an API to access transaction pool information.
type TransactionPoolAPI struct {
	s *SeeleService
}

// NewTransactionPoolAPI creates a new PrivateTransactionPoolAPI object for transaction pool rpc service.
func NewTransactionPoolAPI(s *SeeleService) *TransactionPoolAPI {
	return &TransactionPoolAPI{s}
}
// GetDebtByHash return the debt info by debt hash
func (api *TransactionPoolAPI) GetDebtByHash(debtHash string) (map[string]interface{}, error) {
	hashByte, err := hexutil.HexToBytes(debtHash)
	if err != nil {
		return nil, err
	}
	hash := common.BytesToHash(hashByte)

	output := make(map[string]interface{})
	debt := api.s.DebtPool().GetDebtByHash(hash)
	if debt != nil {
		output["debt"] = debt
		output["status"] = "pool"

		return output, nil
	}

	store := api.s.chain.GetStore()
	debtIndex, err := store.GetDebtIndex(hash)
	if err != nil {
		api.s.log.Info(err.Error())
		return nil, api2.ErrDebtNotFound
	}

	if debtIndex != nil {
		block, err := store.GetBlock(debtIndex.BlockHash)
		if err != nil {
			return nil, err
		}

		output["debt"] = block.Debts[debtIndex.Index]
		output["status"] = "block"
		output["blockHash"] = block.HeaderHash.ToHex()
		output["blockHeight"] = block.Header.Height
		output["debtIndex"] = debtIndex

		return output, nil
	}

	return nil, nil
}
