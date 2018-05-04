/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import "github.com/seeleteam/go-seele/common"

// Receipt represents the contract processing receipt.
type Receipt struct {
	Result          []byte
	PostState       common.Hash // Trie root hash after tx processed.
	Logs            []*Log
	TxHash          common.Hash
	ContractAddress common.Address // Used when tx (nil To address) is to create a contract.
}
