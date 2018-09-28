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
		Result:          []byte("test"),
		Failed:          false,
		UsedGas:         uint64(0),
		PostState:       common.EmptyHash,
		Logs:            []*types.Log{},
		TxHash:          common.EmptyHash,
		ContractAddress: []byte("test"),
		TotalFee:        uint64(0),
	}
	return &receipt
}

func Test_OdrReceipt_Serializable(t *testing.T) {
	// with nil receipt
	request := odrReceipt{
		OdrItem: OdrItem{
			ReqID: 38,
			Error: "hello",
		},
		TxHash: common.StringToHash("tx hash"),
	}

	assertSerializable(t, &request, &odrReceipt{})

	// with receipt
	request = odrReceipt{
		OdrItem: OdrItem{
			ReqID: 38,
			Error: "hello",
		},
		TxHash:  common.StringToHash("tx hash"),
		Receipt: newTestReceipt(),
	}

	assertSerializable(t, &request, &odrReceipt{})
}
