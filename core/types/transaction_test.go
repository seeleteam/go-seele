/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package types

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

func randomAccount(t *testing.T) (*ecdsa.PrivateKey, common.Address) {
	privKey, keyErr := crypto.GenerateKey()
	if keyErr != nil {
		t.Fatalf("Failed to generate ECDSA private key, error = %s", keyErr.Error())
	}

	hexAddress := crypto.PubkeyToString(&privKey.PublicKey)

	return privKey, common.HexMustToAddres(hexAddress)
}

func randomAddress(t *testing.T) common.Address {
	_, address := randomAccount(t)
	return address
}

func newTestTx(t *testing.T, amount int64, nonce uint64, sign bool) *Transaction {
	fromPrivKey, fromAddress := randomAccount(t)
	toAddress := randomAddress(t)

	tx := NewTransaction(fromAddress, toAddress, big.NewInt(amount), nonce)

	if sign {
		tx.Sign(fromPrivKey)
	}

	return tx
}

type mockStateDB struct {
	balances map[common.Address]*big.Int
	nonces   map[common.Address]uint64
}

func (db *mockStateDB) GetBalance(address common.Address) *big.Int {
	if balance, found := db.balances[address]; found {
		return balance
	}

	return big.NewInt(0)
}

func (db *mockStateDB) GetNonce(address common.Address) uint64 {
	if nonce, found := db.nonces[address]; found {
		return nonce
	}

	return 0
}

func newTestStateDB(address common.Address, nonce, balance uint64) stateDB {
	return &mockStateDB{
		balances: map[common.Address]*big.Int{address: new(big.Int).SetUint64(balance)},
		nonces:   map[common.Address]uint64{address: nonce},
	}
}

// Validate successfully if no data changed.
func Test_Transaction_Validate_NoDataChange(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	statedb := newTestStateDB(tx.Data.From, 38, 200)
	err := tx.Validate(statedb)
	assert.Equal(t, err, error(nil))
}

// Validate failed if transaction not signed.
func Test_Transaction_Validate_NotSigned(t *testing.T) {
	tx := newTestTx(t, 100, 38, false)
	statedb := newTestStateDB(tx.Data.From, 38, 200)
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrSigMissing)
}

// Validate failed if transaction Hash value changed.
func Test_Transaction_Validate_HashChanged(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	tx.Hash = crypto.HashBytes([]byte("test"))
	statedb := newTestStateDB(tx.Data.From, 38, 200)
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrHashMismatch)
}

// Validate failed if transaction data changed.
func Test_Transaction_Validate_TxDataChanged(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	tx.Data.Amount.SetInt64(200)
	statedb := newTestStateDB(tx.Data.From, 38, 200)
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrHashMismatch)
}

// Validate failed if transaction data changed along with Hash updated.
func Test_Transaction_Validate_SignInvalid(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)

	// Change amount and update Hash in transaction.
	tx.Data.Amount.SetInt64(200)
	tx.Hash = crypto.MustHash(tx.Data)

	statedb := newTestStateDB(tx.Data.From, 38, 200)
	err := tx.Validate(statedb)

	assert.Equal(t, err, ErrSigInvalid)
}

func Test_MerkleRootHash_Empty(t *testing.T) {
	hash := MerkleRootHash(nil)
	assert.Equal(t, hash, emptyTxRootHash)
}

func Test_Transaction_Validate_BalanceNotEnough(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	statedb := newTestStateDB(tx.Data.From, 38, 50)
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrBalanceNotEnough)
}

func Test_Transaction_Validate_NonceTooLow(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	statedb := newTestStateDB(tx.Data.From, 40, 200)
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrNonceTooLow)
}

func Test_Transaction_Validate_PayloadOversized(t *testing.T) {
	from := crypto.MustGenerateRandomAddress()
	to := crypto.MustGenerateRandomAddress()

	// Cannot create a tx with oversized payload.
	tx, err := NewMessageTransaction(*from, *to, big.NewInt(100), 38, make([]byte, MaxPayloadSize+1))
	assert.Equal(t, err, ErrPayloadOversized)

	// Create a tx with valid payload
	tx, err = NewMessageTransaction(*from, *to, big.NewInt(100), 38, []byte("hello"))
	assert.Equal(t, err, error(nil))
	tx.Data.Payload = make([]byte, MaxPayloadSize+1) // modify the payload to invalid size.

	statedb := newTestStateDB(tx.Data.From, 38, 200)

	err = tx.Validate(statedb)
	assert.Equal(t, err, ErrPayloadOversized)
}
