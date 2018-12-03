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
	"github.com/seeleteam/go-seele/crypto"
)

type odrDebtRequest struct {
	OdrItem
	DebtHash common.Hash
}

type odrDebtResponse struct {
	OdrProvableResponse
	Debt *types.Debt `rlp:"nil"`
}

func (req *odrDebtRequest) code() uint16 {
	return debtRequestCode
}

func (req *odrDebtRequest) handle(lp *LightProtocol) (uint16, odrResponse) {
	debt, index, err := api.GetDebt(lp.debtPool, lp.chain.GetStore(), req.DebtHash)
	if err != nil {
		err = errors.NewStackedErrorf(err, "failed to get debt by hash %v", req.DebtHash)
		return newErrorResponse(debtResponseCode, req.ReqID, err)
	}

	response := &odrDebtResponse{
		OdrProvableResponse: OdrProvableResponse{
			OdrItem: OdrItem{
				ReqID: req.ReqID,
			},
			BlockIndex: index,
		},
		Debt: debt,
	}

	// debt is still in pool.
	if response.Debt == nil || response.BlockIndex == nil {
		return debtResponseCode, response
	}

	block, err := lp.chain.GetStore().GetBlock(response.BlockIndex.BlockHash)
	if err != nil {
		err = errors.NewStackedErrorf(err, "failed to get block by hash %v", response.BlockIndex.BlockHash)
		return newErrorResponse(debtResponseCode, req.ReqID, err)
	}

	debtTrie := types.GetDebtTrie(block.Debts)
	proof, err := debtTrie.GetProof(req.DebtHash.Bytes())
	if err != nil {
		err = errors.NewStackedError(err, "failed to get debt trie proof")
		return newErrorResponse(debtResponseCode, req.ReqID, err)
	}

	response.Proof = mapToArray(proof)

	return debtResponseCode, response
}

func (response *odrDebtResponse) validateUnpackedDebt(debtHash common.Hash) error {
	if response.Debt == nil {
		return nil
	}

	if !debtHash.Equal(response.Debt.Hash) {
		return types.ErrHashMismatch
	}

	if !debtHash.Equal(crypto.MustHash(response.Debt.Data)) {
		return types.ErrHashMismatch
	}

	return nil
}

func (response *odrDebtResponse) validate(request odrRequest, bcStore store.BlockchainStore) error {
	header, err := response.proveHeader(bcStore)
	if err != nil {
		return errors.NewStackedError(err, "failed to prove block header")
	}

	debtHash := request.(*odrDebtRequest).DebtHash

	// debt not packed yet.
	if header == nil {
		return response.validateUnpackedDebt(debtHash)
	}

	response.Debt = new(types.Debt)
	if err = response.proveMerkleTrie(header.DebtHash, debtHash.Bytes(), response.Debt); err != nil {
		return errors.NewStackedError(err, "failed to prove merkle trie")
	}

	return nil
}
