/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"encoding/json"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
)

// Log represents the contract execution log.
type Log struct {
	// Consensus fields:
	// address of the contract that generated the event
	Address common.Address
	// list of topics provided by the contract.
	Topics []common.Hash
	// supplied by the contract, usually ABI-encoded
	Data []byte
	// Derived fields. These fields are filled in by the node
	// but not secured by consensus.
	// block in which the transaction was included
	BlockNumber uint64
	// index of the transaction in the block
	TxIndex uint
}

// MarshalJSON marshal in hex string instead of base64
func (log *Log) MarshalJSON() ([]byte, error) {
	var o struct {
		Address     string   `json:"address" gencodec:"required"`
		Topics      []string `json:"topics" gencodec:"required"`
		Data        string   `json:"data" gencodec:"required"`
		BlockNumber uint64   `json:"blockNumber"`
		TxIndex     uint     `json:"transactionIndex" gencodec:"required"`
	}

	o.Address = log.Address.Hex()
	topics := make([]string, len(log.Topics))
	for index, topic := range log.Topics {
		topics[index] = topic.Hex()
	}
	o.Topics = topics
	o.Data = hexutil.BytesToHex(log.Data)
	o.BlockNumber = log.BlockNumber
	o.TxIndex = log.TxIndex
	return json.Marshal(&o)
}
