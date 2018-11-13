/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/trie"
)

// DebtSize debt serialized size
const DebtSize = 118

var DebtDataFlag = []byte{0x01}

var (
	errWrongShardNumber  = errors.New("wrong from shard number")
	errInvalidAccount    = errors.New("invalid account, unexpected shard number")
	errInvalidHash       = errors.New("debt hash is invalid")
	errInvalidFee        = errors.New("debt fee is invalid")
	ErrMsgVerifierFailed = "failed to validate debt via verifier"
)

// DebtData debt data
type DebtData struct {
	TxHash  common.Hash // the hash of the executed transaction
	From    common.Address
	Nonce   uint64
	Account common.Address // debt for account
	Amount  *big.Int       // debt amount
	Price   *big.Int       // debt price
	Code    common.Bytes   // debt contract code
}

// Debt debt class
type Debt struct {
	Hash common.Hash // Debt hash of DebtData
	Data DebtData
}

// DebtVerifier interface
type DebtVerifier interface {
	// ValidateDebt validate debt
	// returns packed whether debt is packed
	// returns confirmed whether debt is confirmed
	// returns retErr error info
	ValidateDebt(debt *Debt) (packed bool, confirmed bool, err error)

	// IfDebtPacked
	// returns packed whether debt is packed
	// returns confirmed whether debt is confirmed
	// returns retErr error info
	IfDebtPacked(debt *Debt) (packed bool, confirmed bool, err error)
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
func (d *Debt) Validate(verifier DebtVerifier, isPool bool, targetShard uint) (recoverable bool, retErr error) {
	if d.Data.From.Shard() == targetShard {
		retErr = errWrongShardNumber
		return
	}

	toShard := d.Data.Account.Shard()
	if toShard != targetShard {
		retErr = fmt.Errorf("invalid account, unexpected shard number, have %d, expected %d", toShard, targetShard)
		return
	}

	if d.Hash != d.Data.Hash() {
		retErr = errInvalidHash
		return
	}

	if d.Data.Price == nil || d.Data.Price.Sign() <= 0 {
		retErr = errInvalidFee
		return
	}

	// validate debt, skip validation when verifier is nil for test
	if verifier != nil {
		packed, confirmed, err := verifier.ValidateDebt(d)
		if packed {
			recoverable = true
		}

		if confirmed {
			return
		}

		if err != nil || !confirmed {
			if (isPool && !packed) || !isPool {
				retErr = errors.NewStackedError(err, ErrMsgVerifierFailed)
			}
		}
	}

	return
}

func (data *DebtData) Hash() common.Hash {
	return crypto.MustHash(data)
}

// Size is the bytes of debt
func (d *Debt) Size() int {
	return DebtSize + len(d.Data.Code)
}

func (d *Debt) FromAccount() common.Address {
	return d.Data.From
}

func (d *Debt) ToAccount() common.Address {
	return d.Data.Account
}

func (d *Debt) Nonce() uint64 {
	return d.Data.Nonce
}

func (d *Debt) Price() *big.Int {
	return d.Data.Price
}

func (d *Debt) GetHash() common.Hash {
	return d.Hash
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

func (d *Debt) Fee() *big.Int {
	// @todo for contract case, should use the fee in tx receipt
	txIntrFee := new(big.Int).Mul(d.Data.Price, new(big.Int).SetUint64(TransferAmountIntrinsicGas*2))

	return GetDebtShareFee(txIntrFee)
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

	data := DebtData{
		TxHash:  tx.Hash,
		From:    tx.Data.From,
		Nonce:   tx.Data.AccountNonce,
		Account: tx.Data.To,
		Amount:  big.NewInt(0).Set(tx.Data.Amount),
		Price:   tx.Data.GasPrice,
		Code:    make([]byte, 0), // @todo init when its a contract tx
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
