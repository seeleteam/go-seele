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

	return privKey, common.HexToAddress(hexAddress)
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

// Validate successfully if no data changed.
func Test_Transaction_Validate_NoDataChange(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	err := tx.Validate()
	assert.Equal(t, err, error(nil))
}

// Validate failed if transaction not signed.
func Test_Transaction_Validate_NotSigned(t *testing.T) {
	tx := newTestTx(t, 100, 38, false)
	assert.Equal(t, tx.Validate(), errSigMissed)
}

// Validate failed if transaction Hash value changed.
func Test_Transaction_Validate_HashChanged(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	tx.Hash = crypto.HashBytes([]byte("test"))
	assert.Equal(t, tx.Validate(), errHashMismatch)
}

// Validate failed if transation data changed.
func Test_Transaction_Validate_TxDataChanged(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)
	tx.Data.Amount.SetInt64(200)
	assert.Equal(t, tx.Validate(), errHashMismatch)
}

// Validate failed if transaction data changed along with Hash updated.
func Test_Transaction_Validate_SignInvalid(t *testing.T) {
	tx := newTestTx(t, 100, 38, true)

	// Change amount and update Hash in transaction.
	tx.Data.Amount.SetInt64(200)
	tx.Hash = crypto.MustHash(tx.Data)

	assert.Equal(t, tx.Validate(), errSigInvalid)
}
