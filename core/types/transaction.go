/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

var (
	errAmountNil      = errors.New("amount is null")
	errAmountNegative = errors.New("amount is negative")
	errHashMismatch   = errors.New("hash mismatch")
	errSigMissed      = errors.New("signature missed")
	errSigInvalid     = errors.New("signature is invalid")

	emptyTxRootHash = txsTrieSum([]*Transaction{})
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

	txDataBytes := common.SerializePanic(txData)
	txDataHash := crypto.Keccak256Hash(txDataBytes)

	return &Transaction{common.BytesToHash(txDataHash), txData, nil}
}

// Sign signs the transaction with private key.
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) {
	txDataBytes := common.SerializePanic(tx.Data)
	txDataHash := crypto.Keccak256Hash(txDataBytes)

	tx.Hash = common.BytesToHash(txDataHash)
	tx.Signature = crypto.NewSignature(privKey, txDataHash)
}

// Validate returns true if the transation is valid, otherwise false.
func (tx *Transaction) Validate() error {
	if tx.Data == nil || tx.Data.Amount == nil {
		return errAmountNil
	}

	if tx.Data.Amount.Sign() < 0 {
		return errAmountNegative
	}

	// TODO tx.Data.Amount <= account.Balance
	// TODO validate tx.Data.AccountNonce against account.Nonce

	if tx.Signature == nil {
		return errSigMissed
	}

	txDataBytes := common.SerializePanic(tx.Data)
	txDataHash := crypto.Keccak256Hash(txDataBytes)
	if !bytes.Equal(txDataHash, tx.Hash.Bytes()) {
		return errHashMismatch
	}

	pubKey := crypto.ToECDSAPub(tx.Data.From.Bytes())
	if !tx.Signature.Verify(pubKey, txDataHash) {
		return errSigInvalid
	}

	return nil
}

// txsTrieSum calculates and returns the transactions trie root hash.
// TODO depend on the merkle tree implementation.
func txsTrieSum(txs []*Transaction) common.Hash {
	txsBytes := make([][]byte, len(txs))

	for i, tx := range txs {
		txsBytes[i] = common.SerializePanic(tx.Data)
	}

	return common.BytesToHash(crypto.Keccak256Hash(txsBytes...))
}
