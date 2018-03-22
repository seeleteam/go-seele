/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import "math/big"

// Account is a balance model for blockchain
type Account struct {
	Nonce  uint64
	Amount *big.Int
}

// stateObject is state object for statedb
type stateObject struct {
	account Account
	dirty   bool
}

func newStateObject() *stateObject {
	return &stateObject{
		account: Account{
			Nonce:  0,
			Amount: new(big.Int),
		},
		dirty: false,
	}
}

// SetNonce set nonce of account
func (s *stateObject) SetNonce(nonce uint64) {
	s.account.Nonce = nonce
	s.dirty = true
}

// GetNonce get nonce of account
func (s *stateObject) GetNonce() uint64 {
	return s.account.Nonce
}

// GetNonce get nonce of account
func (s *stateObject) GetAmount() *big.Int {
	return s.account.Amount
}

// SetAmount set amount of account
func (s *stateObject) SetAmount(amount *big.Int) {
	s.account.Amount = amount
	s.dirty = true
}

// AddAmount add amount of account
func (s *stateObject) AddAmount(amount *big.Int) {
	s.SetAmount(new(big.Int).Add(s.account.Amount, amount))
}

// SubAmount sub amount of account
func (s *stateObject) SubAmount(amount *big.Int) {
	s.SetAmount(new(big.Int).Sub(s.account.Amount, amount))
}
