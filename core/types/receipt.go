/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/trie"
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
	TotalFee        uint64      // the full cost of the transaction
}

// ReceiptIndex represents an index that used to query block info by tx hash.
type ReceiptIndex indexInBlock

// GetReceiptTrie generate trie according the receipts
func GetReceiptTrie(receipts []*Receipt) *trie.Trie {
	emptyTrie, err := trie.NewTrie(common.EmptyHash, make([]byte, 0), nil)
	if err != nil {
		panic(err)
	}

	for _, r := range receipts {
		buff := common.SerializePanic(r)
		emptyTrie.Put(crypto.HashBytes(buff).Bytes(), buff)
	}

	return emptyTrie
}

// ReceiptMerkleRootHash calculates and returns the merkle root hash of the specified receipts.
// If the given receipts are empty, return empty hash.
func ReceiptMerkleRootHash(receipts []*Receipt) common.Hash {
	if len(receipts) == 0 {
		return emptyReceiptRootHash
	}

	emptyTrie := GetReceiptTrie(receipts)
	return emptyTrie.Hash()
}

// MakeRewardReceipt generates the receipt for the specified reward transaction
func MakeRewardReceipt(reward *Transaction) *Receipt {
	return &Receipt{
		TxHash: reward.Hash,
	}
}
