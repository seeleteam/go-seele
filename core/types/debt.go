/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/trie"
)

// DebtSize debt serialized size
const DebtSize = 98

// DebtData debt data
type DebtData struct {
	TxHash    common.Hash    // the hash of the executed transaction
	FromShard uint           // tx shard
	Account   common.Address // debt for account
	Amount    *big.Int       // debt amount
	Fee       *big.Int       // debt fee
	Code      common.Bytes   // debt contract code
}

// Debt debt class
type Debt struct {
	Hash common.Hash // Debt hash of DebtData
	Data DebtData
}

// DebtVerifier interface
type DebtVerifier interface {
	// ValidateDebt validate debt
	// returns bool recoverable error
	// returns error error info
	ValidateDebt(debt *Debt) (bool, error)
}

// DebtIndex debt index
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

// Validate validate debt with verifier
// If verifier is nil, will skip it.
// If isPool is true, we don't return error when the error is recoverable
func (d *Debt) Validate(verifier DebtVerifier, isPool bool) error {
	if d.Data.FromShard == common.LocalShardNumber {
		return errors.New("wrong from shard number")
	}

	if d.Data.Account.Shard() == d.Data.FromShard {
		return errors.New("invalid account")
	}

	if d.Hash != d.Data.Hash() {
		return errors.New("wrong hash")
	}

	// validate debt, skip validation when verifier is nil for test
	if verifier != nil {
		ok, err := verifier.ValidateDebt(d)
		if err != nil {
			if (isPool && !ok) || !isPool {
				return fmt.Errorf("validate debt failed, error: %s", err)
			}
		}
	}

	return nil
}

func (data *DebtData) Hash() common.Hash {
	return crypto.MustHash(data)
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
	unit := big.NewInt(0).Div(fee, big.NewInt(2))
	return unit
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

	toShard := tx.Data.To.Shard()
	if withContext && toShard == common.LocalShardNumber {
		return nil
	}

	fromShard := tx.Data.From.Shard()
	if !withContext && fromShard == toShard {
		return nil
	}

	// @todo for contract case, should use the fee in tx receipt
	txIntrFee := new(big.Int).Mul(tx.Data.GasPrice, new(big.Int).SetUint64(TransferAmountIntrinsicGas*2))

	data := DebtData{
		TxHash:    tx.Hash,
		FromShard: fromShard,
		Account:   tx.Data.To,
		Amount:    big.NewInt(0).Set(tx.Data.Amount),
		Fee:       GetDebtShareFee(txIntrFee),
		Code:      make([]byte, 0),
	}

	if tx.Data.To.IsEVMContract() {
		data.Code = tx.Data.Payload
	}

	debt := &Debt{
		Data: data,
		Hash: data.Hash(),
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
			shard := d.Data.Account.Shard()
			debts[shard] = append(debts[shard], d)
		}
	}

	return debts
}
