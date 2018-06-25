/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/merkle"
)

var emptyReceiptRootHash = crypto.MustHash("empty receipt root hash")

// Receipt represents the transaction processing receipt.
type Receipt struct {
	Result          []byte      // the execution result of the tx
	Failed          bool        // indicates if execution failed
	UsedGas         uint64      // tx used gas
	PostState       common.Hash // the root hash of the state trie after the tx is processed.
	Logs            []*Log      // the log objects
	TxHash          common.Hash // the hash of the executed transaction
	ContractAddress []byte      // Used when the tx (nil To address) is to create a contract.
}

// CalculateHash calculates and returns the receipt hash.
// This is to implement the merkle.Content interface.
func (receipt *Receipt) CalculateHash() common.Hash {
	return crypto.MustHash(receipt)
}

// Equals indicates if the receipt is equal to the specified content.
// This is to implement the merkle.Content interface.
func (receipt *Receipt) Equals(other merkle.Content) bool {
	otherReceipt, ok := other.(*Receipt)
	if !ok {
		return false
	}

	hash := receipt.CalculateHash()
	otherHash := otherReceipt.CalculateHash()

	return hash.Equal(otherHash)
}

// ReceiptMerkleRootHash calculates and returns the merkle root hash of the specified receipts.
// If the given receipts are empty, return empty hash.
func ReceiptMerkleRootHash(receipts []*Receipt) common.Hash {
	if len(receipts) == 0 {
		return emptyReceiptRootHash
	}

	contents := make([]merkle.Content, len(receipts))
	for i, receipt := range receipts {
		contents[i] = receipt
	}

	bmt, _ := merkle.NewTree(contents)

	return bmt.MerkleRoot()
}

// MakeRewardReceipt generates the receipt for the specified reward transaction
func MakeRewardReceipt(reward *Transaction) *Receipt {
	return &Receipt{
		TxHash: reward.Hash,
	}
}
