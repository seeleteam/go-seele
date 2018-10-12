/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"errors"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

var (
	// ErrBlockHeaderNil is returned when the block header is nil.
	ErrBlockHeaderNil = errors.New("block header is nil")

	// ErrBlockHashMismatch is returned when the block hash does not match the header hash.
	ErrBlockHashMismatch = errors.New("block header hash mismatch")

	// ErrBlockTxsHashMismatch is returned when the block transactions hash does not match
	// the transaction root hash in the header.
	ErrBlockTxsHashMismatch = errors.New("block transactions root hash mismatch")

	// ErrBlockTxDebtHashMismatch is returned when the calculated tx debts hash of block
	// does not match the debts root hash in block header.
	ErrBlockTxDebtHashMismatch = errors.New("block transaction debts hash mismatch")

	// ErrBlockDebtHashMismatch is returned when the calculated debts hash of block
	// does not match the debts root hash in block header.
	ErrBlockDebtHashMismatch = errors.New("block debts hash mismatch")
)

// BlockHeader represents the header of a block in the blockchain.
type BlockHeader struct {
	PreviousBlockHash common.Hash    // PreviousBlockHash represents the hash of the parent block
	Creator           common.Address // Creator is the coinbase of the miner which mined the block
	StateHash         common.Hash    // StateHash is the root hash of the state trie
	TxHash            common.Hash    // TxHash is the root hash of the transaction merkle tree
	ReceiptHash       common.Hash    // ReceiptHash is the root hash of the receipt merkle tree
	TxDebtHash        common.Hash    // TxDebtHash is the root hash of the tx's debt merkle tree
	DebtHash          common.Hash    // DebtHash is the root hash of the debt merkle tree
	Difficulty        *big.Int       // Difficulty is the difficulty of the block
	Height            uint64         // Height is the number of the block
	CreateTimestamp   *big.Int       // CreateTimestamp is the timestamp when the block is created
	Witness           []byte         //Witness is the block pow proof info
	ExtraData         []byte         // ExtraData stores the extra info of block header.
}

// Clone returns a clone of the block header.
func (header *BlockHeader) Clone() *BlockHeader {
	clone := *header

	if clone.Difficulty = new(big.Int); header.Difficulty != nil {
		clone.Difficulty.Set(header.Difficulty)
	}

	if clone.CreateTimestamp = new(big.Int); header.CreateTimestamp != nil {
		clone.CreateTimestamp.Set(header.CreateTimestamp)
	}

	clone.ExtraData = common.CopyBytes(header.ExtraData)
	clone.Witness = common.CopyBytes(header.Witness)

	return &clone
}

// Hash calculates and returns the hash of the bloch header.
func (header *BlockHeader) Hash() common.Hash {
	return crypto.MustHash(header)
}

// Block represents a block in the blockchain.
type Block struct {
	HeaderHash   common.Hash    // HeaderHash is the hash of the RLP encoded header bytes
	Header       *BlockHeader   // Header is the block header, a block header is about 165byte
	Transactions []*Transaction // Transactions is the block payload
	Debts        []*Debt        // Debts for cross shard transaction
}

// NewBlock creates a new block. The input header is copied so that
// any change will not affect the block. The input transaction
// array is copied, but each transaction is not copied.
// So any change of the input transaction will affect the block.
// The input receipt array is the same behavior with transation array.
func NewBlock(header *BlockHeader, txs []*Transaction, receipts []*Receipt, debts []*Debt) *Block {
	block := &Block{
		Header: header.Clone(),
	}

	// Copy the transactions and update the transaction trie root hash.
	block.Header.TxHash = MerkleRootHash(txs)
	if len(txs) > 0 {
		block.Transactions = make([]*Transaction, len(txs))
		copy(block.Transactions, txs)
	}

	if len(debts) > 0 {
		block.Debts = make([]*Debt, len(debts))
		copy(block.Debts, debts)
	}

	block.Header.ReceiptHash = ReceiptMerkleRootHash(receipts)
	block.Header.DebtHash = DebtMerkleRootHash(debts)
	block.Header.TxDebtHash = DebtMerkleRootHash(NewDebts(txs))

	// Calculate the block header hash.
	block.HeaderHash = block.Header.Hash()

	return block
}

// GetExcludeRewardTransactions returns all txs of a block except for the reward transaction
func (block *Block) GetExcludeRewardTransactions() []*Transaction {
	if len(block.Transactions) == 0 {
		return block.Transactions
	}

	return block.Transactions[1:]
}

// FindTransaction returns the transaction of the specified hash if found. Otherwise, it returns nil.
func (block *Block) FindTransaction(txHash common.Hash) *Transaction {
	for _, tx := range block.Transactions {
		if tx.Hash == txHash {
			return tx
		}
	}

	return nil
}

// GetShardNumber returns the shard number of the block, which means the shard number of the creator.
func (block *Block) GetShardNumber() uint {
	if block.Header == nil {
		return common.UndefinedShardNumber
	}

	return block.Header.Creator.Shard()
}

// Validate validates state independent fields in a block.
func (block *Block) Validate() error {
	// Block must have header
	if block.Header == nil {
		return ErrBlockHeaderNil
	}

	// Validate block header hash
	if !block.HeaderHash.Equal(block.Header.Hash()) {
		return ErrBlockHashMismatch
	}

	// Validate tx merkle root hash
	if h := MerkleRootHash(block.Transactions); !h.Equal(block.Header.TxHash) {
		return ErrBlockTxsHashMismatch
	}

	// Validates debt root hash.
	if h := DebtMerkleRootHash(NewDebts(block.Transactions)); !h.Equal(block.Header.TxDebtHash) {
		return ErrBlockTxDebtHashMismatch
	}

	if h := DebtMerkleRootHash(block.Debts); !h.Equal(block.Header.DebtHash) {
		return ErrBlockDebtHashMismatch
	}

	return nil
}
