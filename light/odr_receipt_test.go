/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/stretchr/testify/assert"
)

func newTestReceipt() *types.Receipt {
	receipt := types.Receipt{
		Result:          []byte("test"),
		Failed:          false,
		UsedGas:         uint64(0),
		PostState:       common.EmptyHash,
		Logs:            nil,
		TxHash:          common.EmptyHash,
		ContractAddress: []byte("test"),
		TotalFee:        uint64(0),
	}
	return &receipt
}

func newTestOdrtReceipt(receipt *types.Receipt) *odrtReceipt {
	odrtReceipt := odrtReceipt{
		TxHash:  common.EmptyHash,
		Receipt: receipt,
		odrItem: odrItem{
			ReqID: 0,
			Error: "",
		},
	}

	return &odrtReceipt
}

func Test_handleResponse(t *testing.T) {
	odrtReceiptRequest := newTestOdrtReceipt(&types.Receipt{})
	receipt := newTestReceipt()
	odrtReceiptResponse := newTestOdrtReceipt(receipt)

	odrtReceiptRequest.handleResponse(odrtReceiptResponse)

	assert.Equal(t, odrtReceiptRequest.Receipt == odrtReceiptResponse.Receipt, true)
	assert.Equal(t, odrtReceiptRequest.Error == odrtReceiptResponse.Error, true)
}
