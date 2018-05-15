/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"github.com/seeleteam/go-seele/common"
)

// Log represents the contract execution log.
type Log struct {
	// Consensus fields:
	// address of the contract that generated the event
	Address common.Address `json:"address" gencodec:"required"`
	// list of topics provided by the contract.
	Topics []common.Hash `json:"topics" gencodec:"required"`
	// supplied by the contract, usually ABI-encoded
	Data []byte `json:"data" gencodec:"required"`

	// Derived fields. These fields are filled in by the node
	// but not secured by consensus.
	// block in which the transaction was included
	BlockNumber uint64 `json:"blockNumber"`
	// index of the transaction in the block
	TxIndex uint `json:"transactionIndex" gencodec:"required"`
}
