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

var (
	errAccountNotFound  = errors.New("account not found")
	errAmountNil        = errors.New("amount is null")
	errAmountNegative   = errors.New("amount is negative")
	errBalanceNotEnough = errors.New("balance not enough")
	errNonceTooLow      = errors.New("nonce too low")
	errHashMismatch     = errors.New("hash mismatch")
	errSigMissed        = errors.New("signature missed")
	errSigInvalid       = errors.New("signature is invalid")

	emptyTxRootHash = crypto.MustHash("empty transaction root hash")
)

// TransactionData wraps the data in a transaction.
type TransactionData struct {
	From         common.Address
	To           *common.Address // nil for contract creation transaction.
	Amount       *big.Int
	AccountNonce uint64
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
	if amount == nil {
		panic("Failed to create tx, amount is nil.")
	}

	if amount.Sign() < 0 {
		panic("Failed to create tx, amount is negative.")
	}

	txData := &TransactionData{
		From:         from,
		To:           &to,
		Amount:       new(big.Int),
		AccountNonce: nonce,
	}

	txData.Amount.Set(amount)

	return &Transaction{crypto.MustHash(txData), txData, nil}
}

// Sign signs the transaction with private key.
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) {
	tx.Hash = crypto.MustHash(tx.Data)
	tx.Signature = crypto.NewSignature(privKey, tx.Hash.Bytes())
}

// Validate returns true if the transation is valid, otherwise false.
func (tx *Transaction) Validate(statedb *state.Statedb) error {
	if tx.Data == nil || tx.Data.Amount == nil {
		return errAmountNil
	}

	if tx.Data.Amount.Sign() < 0 {
		return errAmountNegative
	}

	balance, found := statedb.GetAmount(tx.Data.From)
	if !found {
		return errAccountNotFound
	}

	if tx.Data.Amount.Cmp(balance) > 0 {
		return errBalanceNotEnough
	}

	accountNonce, found := statedb.GetNonce(tx.Data.From)
	if !found {
		return errAccountNotFound
	}

	if tx.Data.AccountNonce < accountNonce {
		return errNonceTooLow
	}

	if tx.Signature == nil {
		return errSigMissed
	}

	txDataHash := crypto.MustHash(tx.Data)
	if !txDataHash.Equal(tx.Hash) {
		return errHashMismatch
	}

	if !tx.Signature.Verify(&tx.Data.From, txDataHash.Bytes()) {
		return errSigInvalid
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
