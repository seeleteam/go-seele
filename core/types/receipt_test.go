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
		Result:    []byte("result"),
		PostState: common.StringToHash("post state"),
		Logs:      []*Log{&Log{}, &Log{}, &Log{}},
		TxHash:    common.StringToHash("tx hash"),
	}
}

func Test_Receipt_CalculateHash(t *testing.T) {
	validReceipt := newTestReceipt()
	emptyReceipt := &Receipt{}

	hash1 := validReceipt.CalculateHash()
	hash2 := emptyReceipt.CalculateHash()

	assert.Equal(t, hash1 != common.EmptyHash, true)
	assert.Equal(t, hash2 != common.EmptyHash, true)
	assert.Equal(t, hash1.ToHex(), "0x19290c5d830478a0c44e7606f1a65a2f649e35ba7bb751e140d446bfd02deae9")
	assert.Equal(t, hash2.ToHex(), "0x731b216e25fa1b08a94ef5548db2045ed436fb4ab1df6a5c3f177bc4792216ef")
}

func Test_Receipt_Equals(t *testing.T) {
	r1 := newTestReceipt()
	r2 := newTestReceipt()

	assert.Equal(t, r1.Equals(r1), true)
	assert.Equal(t, r2.Equals(r2), true)
	assert.Equal(t, r1.Equals(r2), true)

	// change result
	r2.Result = []byte("result2")
	assert.Equal(t, r1.Equals(r2), false)

	// change hash
	r2.Result = r1.Result
	r2.TxHash = common.EmptyHash
	assert.Equal(t, r1.Equals(r2), false)

	// empty receipt
	emptyReceipt := &Receipt{}
	assert.Equal(t, emptyReceipt.Equals(emptyReceipt), true)
	assert.Equal(t, r1.Equals(emptyReceipt), false)
	assert.Equal(t, r2.Equals(emptyReceipt), false)

	// other types of object
	tx := newTestTx(t, 1, 2, 3, true)
	assert.Equal(t, r1.Equals(tx), false)
	assert.Equal(t, r2.Equals(tx), false)
	assert.Equal(t, emptyReceipt.Equals(tx), false)

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
