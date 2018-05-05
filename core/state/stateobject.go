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

// StateObject is the state object for statedb
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

// GetCopy gets a copy of the state object
func (s *StateObject) GetCopy() *StateObject {
	return &StateObject{
		account: Account{
			Nonce:  s.account.Nonce,
			Amount: big.NewInt(0).Set(s.account.Amount),
		},
		dirty: s.dirty,
	}
}

// SetNonce sets the nonce of the account in the state object
func (s *StateObject) SetNonce(nonce uint64) {
	s.account.Nonce = nonce
	s.dirty = true
}

// GetNonce gets the nonce of the account in the state object
func (s *StateObject) GetNonce() uint64 {
	return s.account.Nonce
}

// GetAmount gets the balance amount of the account in the state object
func (s *StateObject) GetAmount() *big.Int {
	return new(big.Int).Set(s.account.Amount)
}

// SetAmount sets the balance amount of the account in the state object
func (s *StateObject) SetAmount(amount *big.Int) {
	if amount.Sign() >= 0 {
		s.account.Amount.Set(amount)
		s.dirty = true
	}
}

// AddAmount adds the specified amount to the balance of the account in the state object
func (s *StateObject) AddAmount(amount *big.Int) {
	s.SetAmount(new(big.Int).Add(s.account.Amount, amount))
}

// SubAmount substracts the specified amount from the balance of the account in the state object
func (s *StateObject) SubAmount(amount *big.Int) {
	s.SetAmount(new(big.Int).Sub(s.account.Amount, amount))
}
