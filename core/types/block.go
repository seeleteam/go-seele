/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

// BlockHeader represents the header of a block in the blockchain.
type BlockHeader struct {
	PreviousBlockHash common.Hash
	Creator           common.Address
	TxHash            common.Hash // Transaction tree root hash
	Difficulty        *big.Int    // Mining difficulty of current block
	Height            uint64
	CreateTimestamp   *big.Int
	Nonce             uint64
}

// Clone returns a clone of block header.
func (header *BlockHeader) Clone() *BlockHeader {
	clone := *header

	if clone.Difficulty = new(big.Int); header.Difficulty != nil {
		clone.Difficulty.Set(header.Difficulty)
	}

	if clone.CreateTimestamp = new(big.Int); header.CreateTimestamp != nil {
		clone.CreateTimestamp.Set(header.CreateTimestamp)
	}

	return &clone
}

// Hash calculates and returns the hash of bloch header.
func (header *BlockHeader) Hash() common.Hash {
	headerBytes := common.SerializePanic(header)
	headerHash := crypto.Keccak256Hash(headerBytes)
	return common.BytesToHash(headerHash)
}

// Block represents a block in the blockchain.
type Block struct {
	HeaderHash   common.Hash // Hash on RLP encoded header bytes.
	Header       *BlockHeader
	Transactions []*Transaction
}

// NewBlock creates a new block. The input header is copied,
// any change will not affect the block. The input transaction
// array is copied, but each transaction is not copied.
// So any change of the input transaction will affect the block.
func NewBlock(header *BlockHeader, txs []*Transaction) *Block {
	block := &Block{
		Header: header.Clone(),
	}

	// Copy the transactions and update the transaction trie root hash.
	if len(txs) == 0 {
		block.Header.TxHash = emptyTxRootHash
	} else {
		block.Header.TxHash = MerkleRootHash(txs)
		block.Transactions = make([]*Transaction, len(txs))
		copy(block.Transactions, txs)
	}

	// Calculate the block header hash.
	headerBytes := common.SerializePanic(block.Header)
	block.HeaderHash = common.BytesToHash(crypto.Keccak256Hash(headerBytes))

	return block
}

// FindTransaction returns transaction of specified hash if found. Otherwise, returns nil.
func (block *Block) FindTransaction(txHash common.Hash) *Transaction {
	for _, tx := range block.Transactions {
		if tx.Hash == txHash {
			return tx
		}
	}

	return nil
}
