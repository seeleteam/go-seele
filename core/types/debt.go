/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/trie"
)

// DebtSize debt serialized size
const DebtSize = 98

var (
	errWrongShardNumber = errors.New("wrong from shard number")
	errInvalidAccount   = errors.New("invalid account, unexpected shard number")
	errInvalidHash      = errors.New("debt hash is invalid")
	errInvalidFee       = errors.New("debt fee is invalid")
)

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

	// IfDebtPacked
	// return bool whether it is packed
	// return error whether get error when checking
	IfDebtPacked(debt *Debt) (bool, error)
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

// Validate validate debt with verifier
// If verifier is nil, will skip it.
// If isPool is true, we don't return error when the error is recoverable
func (d *Debt) Validate(verifier DebtVerifier, isPool bool, targetShard uint) error {
	if d.Data.FromShard == targetShard {
		return errWrongShardNumber
	}

	if d.Data.Account.Shard() != targetShard {
		return errInvalidAccount
	}

	if d.Hash != d.Data.Hash() {
		return errInvalidHash
	}

	if d.Data.Fee == nil || d.Data.Fee.Sign() <= 0 {
		return errInvalidFee
	}

	// validate debt, skip validation when verifier is nil for test
	if verifier != nil {
		ok, err := verifier.ValidateDebt(d)
		if err != nil {
			if (isPool && !ok) || !isPool {
				return errors.NewStackedError(err, "failed to validate debt via verifier")
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

// NewDebtWithContext new a debt
func NewDebtWithContext(tx *Transaction) *Debt {
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

	// reward transaction
	if tx.Data.From == common.EmptyAddress {
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
		d := NewDebtWithContext(tx)
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
		d := NewDebtWithContext(tx)
		if d != nil {
			shard := d.Data.Account.Shard()
			debts[shard] = append(debts[shard], d)
		}
	}

	return debts
}

// DebtArrayToMap transfer debt array to debt map
func DebtArrayToMap(debts []*Debt) [][]*Debt {
	debtsMap := make([][]*Debt, common.ShardCount+1)

	for _, d := range debts {
		shard := d.Data.Account.Shard()
		debtsMap[shard] = append(debtsMap[shard], d)
	}

	return debtsMap
}
