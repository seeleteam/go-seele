/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
)

func newTestReceipt() *Receipt {
	return &Receipt{
		Result:          []byte("result"),
		PostState:       common.StringToHash("post state"),
		Logs:            []*Log{&Log{}, &Log{}, &Log{}},
		TxHash:          common.StringToHash("tx hash"),
		ContractAddress: common.Address{},
	}
}

func Test_Receipt_Equals(t *testing.T) {
	r1 := newTestReceipt()
	r2 := newTestReceipt()

	assert.Equal(t, r1.Equals(r2), true)

	// change result
	r2.Result = []byte("result2")
	assert.Equal(t, r1.Equals(r2), false)
}

func Test_Receipt_ReceiptMerkleRootHash(t *testing.T) {
	assert.Equal(t, ReceiptMerkleRootHash(nil), emptyReceiptRootHash)

	receipts := []*Receipt{
		newTestReceipt(),
		newTestReceipt(),
		newTestReceipt(),
	}

	if root := ReceiptMerkleRootHash(receipts); root.IsEmpty() {
		t.Fatal()
	}
}
