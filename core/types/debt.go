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
const DebtSize = 98

// DebtData debt data
type DebtData struct {
	TxHash  common.Hash    // the hash of the executed transaction
	Shard   uint           // target shard
	Account common.Address // debt for account
	Amount  *big.Int       // debt amount
	Fee     *big.Int       // debt fee
	Code    common.Bytes   // debt contract code
}

// Debt debt class
type Debt struct {
	Hash common.Hash // Debt hash of DebtData
	Data DebtData
}

// DebtVerifier interface
type DebtVerifier interface {
	ValidateDebt(debt *Debt) (bool, error)
}

// DebtIndex debt index
type DebtIndex indexInBlock

// GetDebtTrie generates a debt trie for the specified debts.
func GetDebtTrie(debts []*Debt) *trie.Trie {
	debtTrie := trie.NewEmptyTrie(make([]byte, 0), nil)

	for _, debt := range debts {
		if debt != nil {
			debtTrie.Put(debt.Hash.Bytes(), common.SerializePanic(debt))
		}
	}

	return debtTrie
}

// DebtMerkleRootHash calculates and returns the merkle root hash of the specified debts.
// If the given receipts are empty, return empty hash.
func DebtMerkleRootHash(debts []*Debt) common.Hash {
	debtTrie := GetDebtTrie(debts)
	return debtTrie.Hash()
}

// Size is the bytes of debt
func (d *Debt) Size() int {
	return DebtSize + len(d.Data.Code)
}

// GetDebtsSize is the bytes of debts
func GetDebtsSize(debts []*Debt) int {
	size := 0
	for _, d := range debts {
		size += d.Size()
	}

	return size
}

// GetDebtShareFee get debt share fee
func GetDebtShareFee(fee *big.Int) *big.Int {
	unit := big.NewInt(0).Div(fee, big.NewInt(10))

	share := big.NewInt(0).Mul(unit, big.NewInt(9))
	return share
}

// NewDebt new a debt
func NewDebt(tx *Transaction) *Debt {
	return newDebt(tx, true)
}

// NewDebtWithoutContext new debt
func NewDebtWithoutContext(tx *Transaction) *Debt {
	return newDebt(tx, false)
}

func newDebt(tx *Transaction, withContext bool) *Debt {
	if tx == nil || tx.Data.To.IsEmpty() || tx.Data.To.IsReserved() {
		return nil
	}

	shard := tx.Data.To.Shard()
	if withContext && shard == common.LocalShardNumber {
		return nil
	}

	if !withContext && tx.Data.From.Shard() == shard {
		return nil
	}

	// @todo for contract case, should use the fee in tx receipt
	txIntrFee := new(big.Int).Mul(tx.Data.GasPrice, new(big.Int).SetUint64(TransferAmountIntrinsicGas))

	data := DebtData{
		TxHash:  tx.Hash,
		Shard:   shard,
		Account: tx.Data.To,
		Amount:  big.NewInt(0).Set(tx.Data.Amount),
		Fee:     GetDebtShareFee(txIntrFee),
		Code:    make([]byte, 0),
	}

	if tx.Data.To.IsEVMContract() {
		data.Code = tx.Data.Payload
	}

	debt := &Debt{
		Data: data,
		Hash: crypto.MustHash(data),
	}

	return debt
}

// NewDebts new debts
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

// NewDebtMap new debt map
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
