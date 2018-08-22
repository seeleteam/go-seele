/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/trie"
)

type DebtData struct {
	TxHash  common.Hash    // the hash of the executed transaction
	Shard   uint           //target shard
	Account common.Address //debt for account
	amount  *big.Int       //debt amount
}

type Debt struct {
	Hash common.Hash

	Data DebtData
}

// DebtMerkleRootHash calculates and returns the merkle root hash of the specified debts.
// If the given receipts are empty, return empty hash.
func DebtMerkleRootHash(debts []*Debt) common.Hash {
	if len(debts) == 0 {
		return common.EmptyHash
	}

	emptyTrie, err := trie.NewTrie(common.EmptyHash, make([]byte, 0), nil)
	if err != nil {
		panic(err)
	}

	for _, d := range debts {
		if d == nil {
			continue
		}

		buff := common.SerializePanic(d)
		emptyTrie.Put(d.Hash.Bytes(), buff)
	}

	return emptyTrie.Hash()
}

func NewDebt(tx *Transaction) *Debt {
	if tx == nil || tx.Data.To == common.EmptyAddress {
		return nil
	}

	shard := tx.Data.To.Shard()
	if shard == common.LocalShardNumber {
		return nil
	}

	data := DebtData{
		TxHash:  tx.Hash,
		Shard:   shard,
		Account: tx.Data.To,
		amount:  big.NewInt(0).Set(tx.Data.Amount),
	}

	debt := &Debt{
		Data: data,
		Hash: crypto.MustHash(data),
	}

	return debt
}
