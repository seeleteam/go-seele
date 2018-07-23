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
	PreviousBlockHash common.Hash    // PreviousBlockHash represents the hash of the parent block
	Creator           common.Address // Creator is the coinbase of the miner which mined the block
	StateHash         common.Hash    // StateHash is the root hash of the state trie
	TxHash            common.Hash    // TxHash is the root hash of the transaction merkle tree
	ReceiptHash       common.Hash    // ReceiptHash is the root hash of the receipt merkle tree
	Difficulty        *big.Int       // Difficulty is the difficulty of the block
	Height            uint64         // Height is the number of the block
	CreateTimestamp   *big.Int       // CreateTimestamp is the timestamp when the block is created
	Nonce             uint64         // Nonce is the pow of the block
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
}

// NewBlock creates a new block. The input header is copied so that
// any change will not affect the block. The input transaction
// array is copied, but each transaction is not copied.
// So any change of the input transaction will affect the block.
// The input receipt array is the same behavior with transation array.
func NewBlock(header *BlockHeader, txs []*Transaction, receipts []*Receipt) *Block {
	block := &Block{
		Header: header.Clone(),
	}

	// Copy the transactions and update the transaction trie root hash.
	block.Header.TxHash = MerkleRootHash(txs)
	if len(txs) > 0 {
		block.Transactions = make([]*Transaction, len(txs))
		copy(block.Transactions, txs)
	}

	block.Header.ReceiptHash = ReceiptMerkleRootHash(receipts)

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
