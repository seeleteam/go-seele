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
	"github.com/seeleteam/go-seele/core/state"
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

func newTestStateDB(accounts map[common.Address]state.Account) *state.Statedb {
	statedb, err := state.NewStatedb(common.EmptyHash, nil)
	if err != nil {
		panic(err)
	}

	for addr, account := range accounts {
		stateObj := statedb.GetOrNewStateObject(addr)
		stateObj.SetAmount(account.Amount)
		stateObj.SetNonce(account.Nonce)
	}

	return statedb
}

// Validate successfully if no data changed.
func Test_Transaction_Validate_NoDataChange(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	statedb := newTestStateDB(map[common.Address]state.Account{
		tx.Data.From: state.Account{38, big.NewInt(200)},
	})
	err := tx.Validate(statedb)
	assert.Equal(t, err, error(nil))
}

// Validate failed if transaction not signed.
func Test_Transaction_Validate_NotSigned(t *testing.T) {
	tx := newTestTx(t, 100, 38, false)
	statedb := newTestStateDB(map[common.Address]state.Account{
		tx.Data.From: state.Account{38, big.NewInt(200)},
	})
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrSigMissed)
}

// Validate failed if transaction Hash value changed.
func Test_Transaction_Validate_HashChanged(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	tx.Hash = crypto.HashBytes([]byte("test"))
	statedb := newTestStateDB(map[common.Address]state.Account{
		tx.Data.From: state.Account{38, big.NewInt(200)},
	})
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrHashMismatch)
}

// Validate failed if transation data changed.
func Test_Transaction_Validate_TxDataChanged(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	tx.Data.Amount.SetInt64(200)
	statedb := newTestStateDB(map[common.Address]state.Account{
		tx.Data.From: state.Account{38, big.NewInt(200)},
	})
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrHashMismatch)
}

// Validate failed if transaction data changed along with Hash updated.
func Test_Transaction_Validate_SignInvalid(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)

	// Change amount and update Hash in transaction.
	tx.Data.Amount.SetInt64(200)
	tx.Hash = crypto.MustHash(tx.Data)

	statedb := newTestStateDB(map[common.Address]state.Account{
		tx.Data.From: state.Account{38, big.NewInt(200)},
	})
	err := tx.Validate(statedb)

	assert.Equal(t, err, ErrSigInvalid)
}

func Test_MerkleRootHash_Empty(t *testing.T) {
	hash := MerkleRootHash(nil)
	assert.Equal(t, hash, emptyTxRootHash)
}

func Test_Transaction_Validate_AccountNotFound(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	statedb := newTestStateDB(map[common.Address]state.Account{
		*tx.Data.To: state.Account{38, big.NewInt(200)},
	})
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrAccountNotFound)
}

func Test_Transaction_Validate_BalanceNotEnough(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	statedb := newTestStateDB(map[common.Address]state.Account{
		tx.Data.From: state.Account{38, big.NewInt(50)},
	})
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrBalanceNotEnough)
}

func Test_Transaction_Validate_NonceTooLow(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	statedb := newTestStateDB(map[common.Address]state.Account{
		tx.Data.From: state.Account{40, big.NewInt(200)},
	})
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

	statedb := newTestStateDB(map[common.Address]state.Account{
		tx.Data.From: state.Account{38, big.NewInt(200)},
	})

	err = tx.Validate(statedb)
	assert.Equal(t, err, ErrPayloadOversized)
}
