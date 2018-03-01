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

func randomAddress(t *testing.T) common.Address {
	address, err := common.GenerateRandomAddress()
	if err != nil {
		t.Errorf("Failed to generate random address, error = %s", err.Error())
	}

	return *address
}

func randomKey(t *testing.T) *ecdsa.PrivateKey {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Errorf("Failed to generate ECDSA private key, error = %s", err.Error())
	}

	return privKey
}

func newTestTx(t *testing.T, amount int64, nonce uint64) *Transaction {
	tx := NewTransaction(randomAddress(t), randomAddress(t), big.NewInt(amount), nonce)
	privKey := randomKey(t)
	tx.Sign(privKey)
	return tx
}

// After Sign, the signature is not nil and can be verified against tx hash.
func Test_Transaction_Sign(t *testing.T) {
	tx := newTestTx(t, 100, 38)

	if tx.Signature == nil {
		t.Error("The signature is nil after Sign.")
	}

	assert.Equal(t, tx.Signature.Verify(tx.Hash.Bytes()), true)
}

// Validate successfully if no data changed.
func Test_Transaction_Validate_NoDataChange(t *testing.T) {
	tx := newTestTx(t, 100, 38)
	err := tx.Validate()
	assert.Equal(t, err, error(nil))
}

// Validate failed if transaction not signed.
func Test_Transaction_Validate_NotSigned(t *testing.T) {
	tx := newTestTx(t, 100, 38)
	tx.Signature = nil
	err := tx.Validate()
	assert.Equal(t, err, errSigMissed)
}

// Validate failed if transaction Hash value changed.
func Test_Transaction_Validate_HashChanged(t *testing.T) {
	tx := newTestTx(t, 100, 38)
	tx.Hash = common.BytesToHash(crypto.Keccak256Hash([]byte("test")))
	err := tx.Validate()
	assert.Equal(t, err, errHashMismatch)
}

// Validate failed if transation data changed.
func Test_Transaction_Validate_TxDataChanged(t *testing.T) {
	tx := newTestTx(t, 100, 38)
	tx.Data.Amount.SetInt64(200)
	err := tx.Validate()
	assert.Equal(t, err, errHashMismatch)
}

// Validate failed if transaction data changed along with Hash updated.
func Test_Transaction_Validate_SignInvalid(t *testing.T) {
	tx := newTestTx(t, 100, 38)

	// Change amount and update Hash in transaction.
	tx.Data.Amount.SetInt64(200)
	txDataBytes := common.SerializePanic(tx.Data)
	txDataHash := crypto.Keccak256Hash(txDataBytes)
	tx.Hash = common.BytesToHash(txDataHash)

	err := tx.Validate()
	assert.Equal(t, err, errSigInvalid)
}
