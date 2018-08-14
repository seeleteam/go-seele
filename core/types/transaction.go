/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/merkle"
)

const (
	defaultMaxPayloadSize = 32 * 1024

	// TransactionPreSize is the transaction size excluding payload size
	TransactionPreSize = 169
)

var (
	// ErrAmountNegative is returned when the transaction amount is negative.
	ErrAmountNegative = errors.New("amount is negative")

	// ErrAmountNil is returned when the transation amount is nil.
	ErrAmountNil = errors.New("amount is null")

	// ErrFeeNegative is returned when the transaction fee is negative.
	ErrFeeNegative = errors.New("failed to create tx, fee can't be negative or zero")

	// ErrFeeNil is returned when the transaction fee is nil.
	ErrFeeNil = errors.New("failed to create tx, fee is nil")

	// ErrHashMismatch is returned when the transaction hash and data mismatch.
	ErrHashMismatch = errors.New("hash mismatch")

	// ErrPayloadOversized is returned when the payload size is larger than the MaxPayloadSize.
	ErrPayloadOversized = errors.New("oversized payload")

	// ErrPayloadEmpty is returned when create or call a contract without payload.
	ErrPayloadEmpty = errors.New("empty payload")

	// ErrTimestampMismatch is returned when the timestamp of the miner reward tx doesn't match with the block timestamp.
	ErrTimestampMismatch = errors.New("timestamp mismatch")

	// ErrSigInvalid is returned when the transaction signature is invalid.
	ErrSigInvalid = errors.New("signature is invalid")

	// ErrSigMissing is returned when the transaction signature is missing.
	ErrSigMissing = errors.New("signature missing")

	emptyTxRootHash = common.EmptyHash

	// MaxPayloadSize limits the payload size to prevent malicious transactions.
	MaxPayloadSize = defaultMaxPayloadSize
)

// TransactionData wraps the data in a transaction.
type TransactionData struct {
	From         common.Address // From is the address of the sender
	To           common.Address // To is the receiver address, and empty address is used to create contract
	Amount       *big.Int       // Amount is the amount to be transferred
	AccountNonce uint64         // AccountNonce is the nonce of the sender account
	Fee          *big.Int       // Transaction Fee
	Timestamp    uint64         // Timestamp is used for the miner reward transaction, referring to the block timestamp
	Payload      common.Bytes   // Payload is the extra data of the transaction
}

// Transaction represents a transaction in the blockchain.
type Transaction struct {
	Hash      common.Hash      // Hash is the hash of the transaction data
	Data      TransactionData  // Data is the transaction data
	Signature crypto.Signature // Signature is the signature of the transaction
}

// TxIndex represents an index that used to query block info by tx hash.
type TxIndex struct {
	BlockHash common.Hash
	Index     uint // tx array index in block body
}

type stateDB interface {
	GetBalance(common.Address) *big.Int
	GetNonce(common.Address) uint64
}

// GetTransactionsSize return the transaction size
func GetTransactionsSize(txs []*Transaction) int {
	size := 0
	for _, tx := range txs {
		size += tx.Size()
	}
	return size
}

// Size return the transaction size
func (tx *Transaction) Size() int {
	return TransactionPreSize + len(tx.Data.Payload)
}

// NewTransaction creates a new transaction to transfer asset.
// The transaction data hash is also calculated.
func NewTransaction(from, to common.Address, amount *big.Int, fee *big.Int, nonce uint64) (*Transaction, error) {
	return newTx(from, to, amount, fee, nonce, nil)
}

func newTx(from common.Address, to common.Address, amount *big.Int, fee *big.Int, nonce uint64, payload []byte) (*Transaction, error) {
	txData := TransactionData{
		From:         from,
		To:           to,
		AccountNonce: nonce,
		Payload:      common.CopyBytes(payload),
	}

	if amount != nil {
		txData.Amount = new(big.Int).Set(amount)
	}

	if fee != nil {
		txData.Fee = new(big.Int).Set(fee)
	}

	if txData.Payload == nil {
		txData.Payload = make([]byte, 0)
	}

	tx := &Transaction{
		Data:      txData,
		Signature: crypto.Signature{Sig: make([]byte, 0)},
	}

	if err := tx.ValidateWithoutState(false, true); err != nil {
		return nil, err
	}

	tx.Hash = crypto.MustHash(txData)

	return tx, nil
}

// ValidateWithoutState validates state independent fields in tx.
func (tx Transaction) ValidateWithoutState(signNeeded bool, shardNeeded bool) error {
	// validate amount
	if tx.Data.Amount == nil {
		return ErrAmountNil
	}

	if tx.Data.Amount.Sign() < 0 {
		return ErrAmountNegative
	}

	// validate fee
	if tx.Data.Fee == nil {
		return ErrFeeNil
	}

	if tx.Data.Fee.Sign() <= 0 {
		return ErrFeeNegative
	}

	// validate payload
	if len(tx.Data.Payload) > MaxPayloadSize {
		return ErrPayloadOversized
	}

	if (tx.Data.To.IsEmpty() || tx.Data.To.Type() == common.AddressTypeContract) && len(tx.Data.Payload) == 0 {
		return ErrPayloadEmpty
	}

	// validate shard of from/to address
	if shardNeeded {
		if common.IsShardEnabled() {
			if fromShardNum := tx.Data.From.Shard(); fromShardNum != common.LocalShardNumber {
				return fmt.Errorf("invalid from address, shard number is [%v], but coinbase shard number is [%v]", fromShardNum, common.LocalShardNumber)
			}
		}

		if !tx.Data.To.IsEmpty() && common.IsShardEnabled() {
			if toShardNum := tx.Data.To.Shard(); toShardNum != common.LocalShardNumber {
				return fmt.Errorf("invalid to address, shard number is [%v], but coinbase shard number is [%v]", toShardNum, common.LocalShardNumber)
			}
		}
	}

	// vaildate signature
	if !signNeeded {
		return nil
	}

	if len(tx.Signature.Sig) == 0 {
		return ErrSigMissing
	}

	txDataHash := crypto.MustHash(tx.Data)
	if !txDataHash.Equal(tx.Hash) {
		return ErrHashMismatch
	}

	if !tx.Signature.Verify(tx.Data.From, txDataHash.Bytes()) {
		return ErrSigInvalid
	}

	return nil
}

// NewContractTransaction returns a transaction to create a smart contract.
func NewContractTransaction(from common.Address, amount *big.Int, fee *big.Int, nonce uint64, code []byte) (*Transaction, error) {
	return newTx(from, common.EmptyAddress, amount, fee, nonce, code)
}

// NewMessageTransaction returns a transation with the specified message.
func NewMessageTransaction(from, to common.Address, amount *big.Int, fee *big.Int, nonce uint64, msg []byte) (*Transaction, error) {
	return newTx(from, to, amount, fee, nonce, msg)
}

// NewRewardTransaction creates a reward transaction for the specified miner with the specified reward and block timestamp.
func NewRewardTransaction(miner common.Address, reward *big.Int, timestamp uint64) (*Transaction, error) {
	if reward == nil {
		return nil, ErrAmountNil
	}

	if reward.Sign() < 0 {
		return nil, ErrAmountNegative
	}

	rewardTxData := TransactionData{
		From:      common.Address{},
		To:        miner,
		Amount:    new(big.Int).Set(reward),
		Fee:       big.NewInt(0),
		Timestamp: timestamp,
		Payload:   make([]byte, 0),
	}

	rewardTx := &Transaction{
		Hash:      crypto.MustHash(rewardTxData),
		Data:      rewardTxData,
		Signature: crypto.Signature{Sig: make([]byte, 0)},
	}

	return rewardTx, nil
}

// Sign signs the transaction with the specified private key.
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) {
	tx.Hash = crypto.MustHash(tx.Data)
	tx.Signature = *crypto.MustSign(privKey, tx.Hash.Bytes())
}

// Validate validates all fields in tx.
func (tx *Transaction) Validate(statedb stateDB) error {
	if err := tx.ValidateWithoutState(true, true); err != nil {
		return err
	}

	return tx.ValidateState(statedb)
}

// ValidateState validates state dependent fields in tx.
func (tx *Transaction) ValidateState(statedb stateDB) error {
	consumed := new(big.Int).Add(tx.Data.Amount, tx.Data.Fee)
	if balance := statedb.GetBalance(tx.Data.From); consumed.Cmp(balance) > 0 {
		return fmt.Errorf("balance is not enough, account:%s, balance:%v, amount:%v, fee:%v", tx.Data.From.ToHex(), balance, tx.Data.Amount, tx.Data.Fee)
	}

	if accountNonce := statedb.GetNonce(tx.Data.From); tx.Data.AccountNonce < accountNonce {
		return fmt.Errorf("nonce is too small, acount:%s, tx nonce:%d, state db nonce:%d", tx.Data.From.ToHex(), tx.Data.AccountNonce, accountNonce)
	}

	return nil
}

// CalculateHash calculates and returns the transaction hash.
// This is to implement the merkle.Content interface.
func (tx *Transaction) CalculateHash() common.Hash {
	return crypto.MustHash(tx.Data)
}

// Equals indicates if the transaction is equal to the specified content.
// This is to implement the merkle.Content interface.
func (tx *Transaction) Equals(other merkle.Content) bool {
	otherTx, ok := other.(*Transaction)
	return ok && tx.Hash.Equal(otherTx.Hash)
}

// MerkleRootHash calculates and returns the merkle root hash of the specified transactions.
// If the given transactions are empty, return empty hash.
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

// BatchValidateTxs validates the state independent fields of specified txs in multiple threads.
// Because the signature verification is time consuming (see test Benchmark_Transaction_ValidateWithoutState),
// once a block includes too many txs (e.g. 5000), the txs validation will consume too much time.
func BatchValidateTxs(txs []*Transaction) error {
	len := len(txs)
	threads := runtime.NumCPU() / 4 // in case of CPU 100%

	// single thread for few CPU kernel or few txs to validate.
	if threads <= 1 || len < threads {
		for _, tx := range txs {
			if err := tx.ValidateWithoutState(true, true); err != nil {
				return err
			}
		}

		return nil
	}

	// parallel validates txs
	var err error
	wg := sync.WaitGroup{}

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()

			for j := offset; j < len && err == nil; j += threads {
				if e := txs[j].ValidateWithoutState(true, true); e != nil {
					if err != nil {
						err = e
					}

					break
				}
			}
		}(i)
	}

	wg.Wait()

	return err
}
