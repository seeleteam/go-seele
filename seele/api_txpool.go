/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	api2 "github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
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

	debt, blockIdx, err := api2.GetDebt(api.s.DebtPool(), api.s.chain.GetStore(), hash)
	if err != nil {
		return nil, err
	}

	output := map[string]interface{}{
		"debt": debt,
	}

	if blockIdx == nil {
		output["status"] = "pool"
	} else {
		output["status"] = "block"
		output["blockHash"] = blockIdx.BlockHash.Hex()
		output["blockHeight"] = blockIdx.BlockHeight
		output["debtIndex"] = blockIdx.Index
	}

	return output, nil
}
