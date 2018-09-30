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

func newTestReceipt() []*types.Receipt {
	receipts := []*types.Receipt{
		{
			Result:          []byte("test1"),
			Failed:          false,
			UsedGas:         uint64(0),
			PostState:       common.EmptyHash,
			Logs:            []*types.Log{},
			TxHash:          common.EmptyHash,
			ContractAddress: []byte("test1"),
			TotalFee:        uint64(0),
		},
		{
			Result:          []byte("test2"),
			Failed:          false,
			UsedGas:         uint64(0),
			PostState:       common.EmptyHash,
			Logs:            []*types.Log{},
			TxHash:          common.EmptyHash,
			ContractAddress: []byte("test2"),
			TotalFee:        uint64(0),
		},
		{
			Result:          []byte("test3"),
			Failed:          false,
			UsedGas:         uint64(0),
			PostState:       common.EmptyHash,
			Logs:            []*types.Log{},
			TxHash:          common.EmptyHash,
			ContractAddress: []byte("test3"),
			TotalFee:        uint64(0),
		},
	}
	return receipts
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
		Receipts: newTestReceipt(),
	}

	assertSerializable(t, &request, &odrReceipt{})
}
