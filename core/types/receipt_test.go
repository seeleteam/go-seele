/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/seeleteam/go-seele/common"
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

func Test_Receipt_MakeRewardReceipt(t *testing.T) {
	tx := newTestTx(t, 1, 2, 3, true)
	txHash := tx.Hash
	receipt := MakeRewardReceipt(tx)

	assert.Equal(t, receipt.TxHash, txHash)
}
