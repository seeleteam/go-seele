/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/stretchr/testify/assert"
)

func newTestReceipt() *Receipt {
	return &Receipt{
		Result:    []byte("result"),
		PostState: common.StringToHash("post state"),
		Logs:      []*Log{&Log{}, &Log{}, &Log{}},
		TxHash:    common.StringToHash("tx hash"),
	}
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
