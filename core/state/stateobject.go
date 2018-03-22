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

// StateObject is state object for statedb
type StateObject struct {
	account Account
	dirty   bool
}

func newStateObject() *StateObject {
	return &StateObject{
		account: Account{
			Nonce:  0,
			Amount: new(big.Int),
		},
		dirty: false,
	}
}

// SetNonce set nonce of account
func (s *StateObject) SetNonce(nonce uint64) {
	s.account.Nonce = nonce
	s.dirty = true
}

// GetNonce get nonce of account
func (s *StateObject) GetNonce() uint64 {
	return s.account.Nonce
}

// GetAmount get nonce of account
func (s *StateObject) GetAmount() *big.Int {
	return new(big.Int).Set(s.account.Amount)
}

// SetAmount set amount of account
func (s *StateObject) SetAmount(amount *big.Int) {
	if amount.Sign() >= 0 {
		s.account.Amount.Set(amount)
		s.dirty = true
	}
}

// AddAmount add amount of account
func (s *StateObject) AddAmount(amount *big.Int) {
	s.SetAmount(new(big.Int).Add(s.account.Amount, amount))
}

// SubAmount sub amount of account
func (s *StateObject) SubAmount(amount *big.Int) {
	s.SetAmount(new(big.Int).Sub(s.account.Amount, amount))
}
