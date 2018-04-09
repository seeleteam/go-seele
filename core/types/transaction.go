/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/merkle"
)

const (
	defaultMaxPayloadSize = 32 * 1024
)

var (
	// ErrAccountNotFound is returned when account not found in state DB.
	ErrAccountNotFound = errors.New("account not found")

	// ErrAmountNegative is returned when transaction amount is negative.
	ErrAmountNegative = errors.New("amount is negative")

	// ErrAmountNil is returned when transation amount is nil.
	ErrAmountNil = errors.New("amount is null")

	// ErrBalanceNotEnough is returned when account balance is not enough to transfer to another account.
	ErrBalanceNotEnough = errors.New("balance not enough")

	// ErrHashMismatch is returned when transaction hash and data mismatch.
	ErrHashMismatch = errors.New("hash mismatch")

	// ErrNonceTooLow is returned when transaction nonce is lower than account nonce.
	ErrNonceTooLow = errors.New("nonce too low")

	// ErrPayloadOversized is returned when payload is larger than the MaxPayloadSize.
	ErrPayloadOversized = errors.New("oversized payload")

	// ErrSigInvalid is returned when transaction signature is invalid.
	ErrSigInvalid = errors.New("signature is invalid")

	// ErrSigMissed is returned when transaction signature missed.
	ErrSigMissed = errors.New("signature missed")

	emptyTxRootHash = crypto.MustHash("empty transaction root hash")

	// MaxPayloadSize limits the payload size to prevent malicious transations.
	MaxPayloadSize = defaultMaxPayloadSize
)

// TransactionData wraps the data in a transaction.
type TransactionData struct {
	From         common.Address
	To           *common.Address // nil for contract creation transaction.
	Amount       *big.Int
	AccountNonce uint64
	Payload      []byte
}

// Transaction represents a transaction in the blockchain.
type Transaction struct {
	Hash      common.Hash // hash on transaction data
	Data      *TransactionData
	Signature *crypto.Signature
}

// NewTransaction creates a new transaction to transfer asset.
// The transaction data hash is also calculated.
// Panics if the amount is nil or negative.
func NewTransaction(from, to common.Address, amount *big.Int, nonce uint64) *Transaction {
	tx, _ := newTx(from, &to, amount, nonce, nil)
	return tx
}

func newTx(from common.Address, to *common.Address, amount *big.Int, nonce uint64, payload []byte) (*Transaction, error) {
	if amount == nil {
		panic("Failed to create tx, amount is nil.")
	}

	if amount.Sign() < 0 {
		panic("Failed to create tx, amount is negative.")
	}

	if len(payload) > MaxPayloadSize {
		return nil, ErrPayloadOversized
	}

	txData := &TransactionData{
		From:         from,
		To:           to,
		Amount:       new(big.Int).Set(amount),
		AccountNonce: nonce,
	}

	if len(payload) > 0 {
		cloned := make([]byte, len(payload))
		copy(cloned, payload)
		txData.Payload = cloned
	} else {
		txData.Payload = make([]byte, 0)
	}

	return &Transaction{crypto.MustHash(txData), txData, nil}, nil
}

// NewContractTransaction returns a transation to create a smart contract.
func NewContractTransaction(from common.Address, amount *big.Int, nonce uint64, code []byte) (*Transaction, error) {
	return newTx(from, nil, amount, nonce, code)
}

// NewMessageTransaction returns a transation with specified message.
func NewMessageTransaction(from, to common.Address, amount *big.Int, nonce uint64, msg []byte) (*Transaction, error) {
	return newTx(from, &to, amount, nonce, msg)
}

// Sign signs the transaction with private key.
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) {
	tx.Hash = crypto.MustHash(tx.Data)
	tx.Signature = crypto.NewSignature(privKey, tx.Hash.Bytes())
}

// Validate returns true if the transation is valid, otherwise false.
func (tx *Transaction) Validate(statedb *state.Statedb) error {
	if tx.Data == nil || tx.Data.Amount == nil {
		return ErrAmountNil
	}

	if tx.Data.Amount.Sign() < 0 {
		return ErrAmountNegative
	}

	balance, found := statedb.GetAmount(tx.Data.From)
	if !found {
		return ErrAccountNotFound
	}

	if tx.Data.Amount.Cmp(balance) > 0 {
		return ErrBalanceNotEnough
	}

	accountNonce, found := statedb.GetNonce(tx.Data.From)
	if !found {
		return ErrAccountNotFound
	}

	if tx.Data.AccountNonce < accountNonce {
		return ErrNonceTooLow
	}

	if len(tx.Data.Payload) > MaxPayloadSize {
		return ErrPayloadOversized
	}

	if tx.Signature == nil {
		return ErrSigMissed
	}

	txDataHash := crypto.MustHash(tx.Data)
	if !txDataHash.Equal(tx.Hash) {
		return ErrHashMismatch
	}

	if !tx.Signature.Verify(&tx.Data.From, txDataHash.Bytes()) {
		return ErrSigInvalid
	}

	return nil
}

// CalculateHash calculates and returns the transaction hash.
// This is to implement the merkle.Content interface.
func (tx *Transaction) CalculateHash() common.Hash {
	return crypto.MustHash(tx.Data)
}

// Equals returns if the transaction is equals to the specified content.
// This is to implement the merkle.Content interface.
func (tx *Transaction) Equals(other merkle.Content) bool {
	otherTx, ok := other.(*Transaction)
	return ok && tx.Hash.Equal(otherTx.Hash)
}

// MerkleRootHash calculates and returns the merkle root hash of the specified transactions.
// If the given transactions is empty, return empty hash.
func MerkleRootHash(txs []*Transaction) common.Hash {
	if len(txs) == 0 {
		return emptyTxRootHash
	}

	contents := make([]merkle.Content, len(txs))
	for i, tx := range txs {
		contents[i] = tx
	}

	bmt, _ := merkle.NewTree(contents)

	return bmt.MerkleRoot()
}
