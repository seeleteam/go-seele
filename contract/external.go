/**
* This a temp file that mock the external APIs used in smart contract component.
* It will be removed when blockchain infrastructure constructed, including account,
* security, RPC, consensus and blockchain storage components.
*
* @file
* @copyright defined in go-seele/LICENSE
 */
package contract

import (
	"github.com/seeleteam/go-seele/crypto"
)

const (
	txTypeContractReg int = iota
	txTypeContractCall
	txTypeContractDel
)

// Account wraps the information of smart contract account.
type Account struct {
	AccountAddress string
	CodeAddress    string
	State          []byte
}

// Transaction wraps the smart contract transaction.
type Transaction struct {
	from    string
	to      string
	txType  int
	payload []byte
	sig     *crypto.Signature
}

// TransactionService is the interface for all transaction related operations.
type TransactionService interface {
	// SendTransaction broadcast the smart contract transaction to P2P network for consensus.
	SendTransaction(tx *Transaction)

	// HandleTransaction handle the received smart contract transaction.
	HandleTransaction(tx *Transaction)
}

// BlockchainService is the interface for blockchain related operations.
type BlockchainService interface {
	// WriteContractAccount write contract account on blockchain.
	WriteContractAccount(account *Account)
	// GetContractAccount returns contract account of the specified address.
	GetContractAccount(address string) *Account

	// WriteCode write code on blockchain.
	WriteCode(codeAddress string, code []byte)
	// GetCode returns code of the specified address.
	GetCode(codeAddress string) []byte

	// WriteTransaction write transaction on blockchain.
	WriteTransaction(tx *Transaction)
}
