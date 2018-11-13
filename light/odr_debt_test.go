/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/trie"
	"github.com/stretchr/testify/assert"
)

func newTestDebt(txHash common.Hash) *types.Debt {
	data := types.DebtData{
		TxHash:  txHash,
		Account: common.EmptyAddress,
		Amount:  big.NewInt(38),
		Price:   big.NewInt(666),
		Code:    make([]byte, 0),
	}

	return &types.Debt{
		Hash: crypto.MustHash(data),
		Data: data,
	}
}

func Test_odrDebtRequest_Serializable(t *testing.T) {
	request := &odrDebtRequest{
		DebtHash: common.StringToHash("debt hash"),
	}

	assertSerializable(t, request, &odrDebtRequest{})
}

func Test_odrDebtResponse_Serializable(t *testing.T) {
	// debt in pool
	response := &odrDebtResponse{
		Debt:  newTestDebt(common.StringToHash("tx hash")),
		Proof: make([]proofNode, 0),
	}

	assertSerializable(t, response, &odrDebtResponse{})

	// debt packed in blockchain
	response = &odrDebtResponse{
		Debt: newTestDebt(common.StringToHash("tx hash")),
		BlockIndex: &api.BlockIndex{
			BlockHash:   common.StringToHash("block hash"),
			BlockHeight: 38,
			Index:       66,
		},
		Proof: []proofNode{
			proofNode{
				Key:   "root",
				Value: []byte{1, 2, 3},
			},
			proofNode{
				Key:   "leaf",
				Value: []byte{3, 4, 5},
			},
		},
	}

	assertSerializable(t, response, &odrDebtResponse{})
}

func Test_odrDebtResponse_Validate_NilDebt(t *testing.T) {
	response := &odrDebtResponse{
		BlockIndex: &api.BlockIndex{
			BlockHash:   common.StringToHash("block"),
			BlockHeight: 38,
			Index:       99,
		},
	}

	assert.Nil(t, response.validate(nil, nil))
}

func Test_odrDebtResponse_Validate_HashMismatch(t *testing.T) {
	// request debt hash mismatch with debt hash
	request := &odrDebtRequest{
		DebtHash: common.StringToHash("777"),
	}

	response := &odrDebtResponse{
		Debt: newTestDebt(common.StringToHash("tx 666")),
	}

	assert.Equal(t, types.ErrHashMismatch, response.validate(request, nil))

	// request debt hash mismatch with debt data hash
	request = &odrDebtRequest{
		DebtHash: response.Debt.Hash,
	}

	response.Debt.Data.Nonce++ // change the debt data

	assert.Equal(t, types.ErrHashMismatch, response.validate(request, nil))
}

func Test_odrDebtResponse_Validate_NilBlockIndex(t *testing.T) {
	response := &odrDebtResponse{
		Debt: newTestDebt(common.StringToHash("tx 666")),
	}

	request := &odrDebtRequest{
		DebtHash: response.Debt.Hash,
	}

	assert.Nil(t, response.validate(request, nil))
}

func Test_odrDebtResponse_Validate(t *testing.T) {
	debts := []*types.Debt{
		newTestDebt(common.StringToHash("tx 1")),
		newTestDebt(common.StringToHash("tx 2")),
		newTestDebt(common.StringToHash("tx 3")),
	}

	request := &odrDebtRequest{
		DebtHash: debts[1].Hash,
	}

	// handle the requesst and generate node proof
	debtTrie := types.GetDebtTrie(debts)
	proof, err := debtTrie.GetProof(request.DebtHash.Bytes())
	assert.Nil(t, err)

	response := &odrDebtResponse{
		Debt: debts[1],
		BlockIndex: &api.BlockIndex{
			BlockHash:   common.StringToHash("block"),
			BlockHeight: 38,
			Index:       77,
		},
		Proof: mapToArray(proof),
	}

	// verify the debt trie proof
	value, err := trie.VerifyProof(debtTrie.Hash(), request.DebtHash.Bytes(), arrayToMap(response.Proof))
	assert.Nil(t, err)

	buff := common.SerializePanic(response.Debt)
	assert.Equal(t, value, buff)
}
