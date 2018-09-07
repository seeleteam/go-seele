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

// DebtSize debt serialized size
const DebtSize = 94

type DebtData struct {
	TxHash  common.Hash    // the hash of the executed transaction
	Shard   uint           // target shard
	Account common.Address // debt for account
	Amount  *big.Int       // debt amount
	Fee     *big.Int       // debt fee
	Code    common.Bytes   // debt contract code
}

type Debt struct {
	Hash common.Hash // Debt hash of DebtData
	Data DebtData
}

type DebtIndex indexInBlock

// DebtMerkleRootHash calculates and returns the merkle root hash of the specified debts.
// If the given receipts are empty, return empty hash.
func DebtMerkleRootHash(debts []*Debt) common.Hash {
	if len(debts) == 0 {
		return common.EmptyHash
	}

	debtTrie, err := trie.NewTrie(common.EmptyHash, make([]byte, 0), nil)
	if err != nil {
		panic(err)
	}

	for _, d := range debts {
		if d == nil {
			continue
		}

		buff := common.SerializePanic(d)
		debtTrie.Put(d.Hash.Bytes(), buff)
	}

	return debtTrie.Hash()
}

func (d *Debt) Size() int {
	return DebtSize
}

func GetDebtShareFee(fee *big.Int) *big.Int {
	unit := big.NewInt(0).Div(fee, big.NewInt(10))

	share := big.NewInt(0).Mul(unit, big.NewInt(9))
	return share
}

func NewDebt(tx *Transaction) *Debt {
	if tx == nil || tx.Data.To.IsEmpty() {
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
		Amount:  big.NewInt(0).Set(tx.Data.Amount),
		Fee:     GetDebtShareFee(tx.Data.Fee),
	}

	if tx.Data.To.IsContract() {
		data.Code = tx.Data.Payload
	}

	debt := &Debt{
		Data: data,
		Hash: crypto.MustHash(data),
	}

	return debt
}

func NewDebts(txs []*Transaction) []*Debt {
	debts := make([]*Debt, 0)

	for _, tx := range txs {
		d := NewDebt(tx)
		if d != nil {
			debts = append(debts, d)
		}
	}

	return debts
}

func NewDebtMap(txs []*Transaction) [][]*Debt {
	debts := make([][]*Debt, common.ShardCount+1)

	for _, tx := range txs {
		d := NewDebt(tx)
		if d != nil {
			debts[d.Data.Shard] = append(debts[d.Data.Shard], d)
		}
	}

	return debts
}
