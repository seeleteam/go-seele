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
	address  common.Address
	addrHash common.Hash

	account      Account
	dirtyAccount bool

	code      []byte // contract code
	dirtyCode bool

	// When a state object is marked assuicided, it will be deleted from the trie when commit the state DB.
	suicided bool
}

func newStateObject(address common.Address) *StateObject {
	return &StateObject{
		address:  address,
		addrHash: crypto.HashBytes(address.Bytes()),
		account: Account{
			Amount: new(big.Int),
		},
	}
}

// GetCopy gets a copy of the state object
func (s *StateObject) GetCopy() *StateObject {
	codeCloned := make([]byte, len(s.code))
	copy(codeCloned, s.code)

	objCloned := *s
	objCloned.account.Amount = big.NewInt(0).Set(s.account.Amount)
	objCloned.code = codeCloned

	return &objCloned
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

func (s *StateObject) loadCode(db database.Database) ([]byte, error) {
	if s.code != nil {
		return s.code, nil
	}

	if s.account.CodeHash.IsEmpty() {
		return nil, nil
	}

	code, err := db.Get(s.getCodeKey())
	if err != nil {
		return nil, err
	}

	s.code = code

	return code, nil
}

func (s *StateObject) getCodeKey() []byte {
	return append(keyPrefixCode, s.addrHash.Bytes()...)
}

func (s *StateObject) setCode(code []byte) {
	s.code = code
	s.dirtyCode = true

	s.account.CodeHash = crypto.HashBytes(code)
	s.dirtyAccount = true
}

func (s *StateObject) serializeCode(batch database.Batch) {
	if s.code != nil {
		batch.Put(s.getCodeKey(), s.code)
	}
}

// empty returns whether the account is considered empty (nonce == amount == 0 and no code).
// This is used during EVM execution.
func (s *StateObject) empty() bool {
	return s.account.Nonce == 0 && s.account.Amount.Sign() == 0 && s.account.CodeHash.IsEmpty()
}
