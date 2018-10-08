/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

func newTestReceipt() *types.Receipt {
	receipt := types.Receipt{
		Result:          []byte("test1"),
		Failed:          false,
		UsedGas:         uint64(0),
		PostState:       common.EmptyHash,
		Logs:            []*types.Log{},
		TxHash:          common.EmptyHash,
		ContractAddress: []byte("test1"),
		TotalFee:        uint64(0),
	}
	return &receipt
}

func Test_OdrReceipt_Serializable(t *testing.T) {
	request := odrReceiptRequest{
		OdrItem: OdrItem{
			ReqID: 38,
			Error: "hello",
		},
		TxHash: common.StringToHash("tx hash"),
	}

	assertSerializable(t, &request, &odrReceiptRequest{})

	// with receipt
	response := odrReceiptResponse{
		OdrItem: OdrItem{
			ReqID: 38,
			Error: "hello",
		},
		ReceiptIndex: &types.ReceiptIndex{
			BlockHash: common.StringToHash("tx hash"),
			Index:     uint(0),
		},
		Receipt: newTestReceipt(),
		Proof:   make([]proofNode, 0),
	}
	assertSerializable(t, &response, &odrReceiptResponse{})

	// without receipt
	response = odrReceiptResponse{
		OdrItem: OdrItem{
			ReqID: 38,
			Error: "hello",
		},
		ReceiptIndex: &types.ReceiptIndex{
			BlockHash: common.StringToHash("tx hash"),
			Index:     uint(0),
		},
		Proof: make([]proofNode, 0),
	}
	assertSerializable(t, &response, &odrReceiptResponse{})
}
