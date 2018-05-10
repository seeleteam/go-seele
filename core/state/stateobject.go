/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
)

var keyPrefixCode = []byte("code")

// Account is a balance model for blockchain
type Account struct {
	Nonce    uint64
	Amount   *big.Int
	CodeHash common.Hash // contract code hash
}

// StateObject is the state object for statedb
type StateObject struct {
	address common.Address
	dbErr   error // dbErr is used for record the database error.

	account      Account
	dirtyAccount bool

	code      []byte // contract code
	dirtyCode bool
}

func newStateObject(address common.Address) *StateObject {
	return &StateObject{
		address: address,
		account: Account{
			Amount: new(big.Int),
		},
	}
}

// GetCopy gets a copy of the state object
func (s *StateObject) GetCopy() *StateObject {
	return &StateObject{
		address: s.address,
		dbErr:   s.dbErr,
		account: Account{
			Nonce:    s.account.Nonce,
			Amount:   big.NewInt(0).Set(s.account.Amount),
			CodeHash: s.account.CodeHash,
		},
		dirtyAccount: s.dirtyAccount,
		code:         s.code,
		dirtyCode:    s.dirtyCode,
	}
}

// SetNonce sets the nonce of the account in the state object
func (s *StateObject) SetNonce(nonce uint64) {
	s.account.Nonce = nonce
	s.dirtyAccount = true
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
		s.dirtyAccount = true
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

func (s *StateObject) loadCode(db database.Database) []byte {
	if s.code != nil {
		return s.code
	}

	if s.account.CodeHash.IsEmpty() {
		return nil
	}

	code, err := db.Get(s.getCodeKey(s.address))
	if err != nil {
		s.dbErr = err
	} else {
		s.code = code
	}

	return s.code
}

func (s *StateObject) getCodeKey(address common.Address) []byte {
	return append(keyPrefixCode, address.Bytes()...)
}

func (s *StateObject) setCode(code []byte) {
	s.code = code
	s.dirtyCode = true

	s.account.CodeHash = crypto.HashBytes(code)
	s.dirtyAccount = true
}

func (s *StateObject) serializeCode(batch database.Batch) {
	if s.code == nil {
		batch.Delete(s.getCodeKey(s.address))
	} else {
		batch.Put(s.getCodeKey(s.address), s.code)
	}
}
