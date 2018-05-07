/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import "github.com/seeleteam/go-seele/common"

// Receipt represents the transaction processing receipt.
type Receipt struct {
	Result          []byte // the execution result of the tx
	PostState       common.Hash // the root hash of the state trie after the tx is processed.
	Logs            []*Log // the log objects
	TxHash          common.Hash // the hash of the executed transaction
	ContractAddress common.Address // Used when the tx (nil To address) is to create a contract.
}
