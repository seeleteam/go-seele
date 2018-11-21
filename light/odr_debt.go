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
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/trie"
)

var (
	errDebtVerifyFailed = errors.New("failed to verify the debt")
	errForkDebt         = errors.New("get a debt from a fork chain")
)

type odrDebtRequest struct {
	OdrItem
	DebtHash common.Hash
}

type odrDebtResponse struct {
	OdrItem
	Debt       *types.Debt     `rlp:"nil"`
	BlockIndex *api.BlockIndex `rlp:"nil"`
	Proof      []proofNode
}

func (req *odrDebtRequest) code() uint16 {
	return debtRequestCode
}

func newOdrDebtErrorResponse(reqID uint32, err error) *odrDebtResponse {
	return &odrDebtResponse{
		OdrItem: OdrItem{
			ReqID: reqID,
			Error: err.Error(),
		},
	}
}

func (req *odrDebtRequest) handle(lp *LightProtocol) (uint16, odrResponse) {
	debt, index, err := api.GetDebt(lp.debtPool, lp.chain.GetStore(), req.DebtHash)
	if err != nil {
		err = errors.NewStackedErrorf(err, "failed to get debt by hash %v", req.DebtHash)
		return debtResponseCode, newOdrDebtErrorResponse(req.ReqID, err)
	}

	response := &odrDebtResponse{
		OdrItem: OdrItem{
			ReqID: req.ReqID,
		},
		Debt:       debt,
		BlockIndex: index,
	}

	// debt is still in pool.
	if response.Debt == nil || response.BlockIndex == nil {
		return debtResponseCode, response
	}

	block, err := lp.chain.GetStore().GetBlock(response.BlockIndex.BlockHash)
	if err != nil {
		err = errors.NewStackedErrorf(err, "failed to get block by hash %v", response.BlockIndex.BlockHash)
		return debtResponseCode, newOdrDebtErrorResponse(req.ReqID, err)
	}

	debtTrie := types.GetDebtTrie(block.Debts)
	proof, err := debtTrie.GetProof(req.DebtHash.Bytes())
	if err != nil {
		err = errors.NewStackedError(err, "failed to get debt trie proof")
		return debtResponseCode, newOdrDebtErrorResponse(req.ReqID, err)
	}

	response.Proof = mapToArray(proof)

	return debtResponseCode, response
}

func (response *odrDebtResponse) validate(request odrRequest, bcStore store.BlockchainStore) error {
	if response.Debt == nil {
		return nil
	}

	// ensure the debt hash matched.
	debtHash := request.(*odrDebtRequest).DebtHash
	if !debtHash.Equal(response.Debt.Hash) {
		return types.ErrHashMismatch
	}

	if !debtHash.Equal(crypto.MustHash(response.Debt.Data)) {
		return types.ErrHashMismatch
	}

	// validate the debt trie proof
	if response.BlockIndex != nil {
		header, err := bcStore.GetBlockHeader(response.BlockIndex.BlockHash)
		if err != nil {
			return errors.NewStackedErrorf(err, "failed to get block header by hash %v", response.BlockIndex.BlockHash)
		}

		blockHash, err := bcStore.GetBlockHash(header.Height)
		if err != nil {
			return errors.NewStackedErrorf(err, "failed to get block hash by height %d", header.Height)
		}
		if !blockHash.Equal(header.Hash()) {
			return errForkDebt
		}

		proof := arrayToMap(response.Proof)
		value, err := trie.VerifyProof(header.DebtHash, debtHash.Bytes(), proof)
		if err != nil {
			return errors.NewStackedError(err, "failed to verify the debt trie proof")
		}

		if buff := common.SerializePanic(response.Debt); !bytes.Equal(buff, value) {
			return errDebtVerifyFailed
		}
	}

	return nil
}
