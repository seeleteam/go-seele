/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package types

import (
	"crypto/ecdsa"
	"math/big"
	"strings"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

func randomAccount(t *testing.T) (*ecdsa.PrivateKey, common.Address) {
	common.IsShardDisabled = true

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

	tx := NewTransaction(fromAddress, toAddress, big.NewInt(amount), big.NewInt(0), nonce)

	if sign {
		tx.Sign(fromPrivKey)
	}

	return tx
}

type mockStateDB struct {
	balances  map[common.Address]*big.Int
	nonces    map[common.Address]uint64
	contracts map[common.Address]common.Address
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

func (db *mockStateDB) AddContract(creatorAddr, contractAddr common.Address) {
	db.contracts[contractAddr] = creatorAddr
}

func (db *mockStateDB) GetContractCreator(contractAddr common.Address) (common.Address, bool) {
	if creator, ok := db.contracts[contractAddr]; ok {
		return creator, true
	}

	return common.Address{}, false
}

func newTestStateDB(address common.Address, nonce, balance uint64) *mockStateDB {
	return &mockStateDB{
		balances:  map[common.Address]*big.Int{address: new(big.Int).SetUint64(balance)},
		nonces:    map[common.Address]uint64{address: nonce},
		contracts: make(map[common.Address]common.Address),
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
	tx, err := NewMessageTransaction(*from, *to, big.NewInt(100), big.NewInt(0), 38, make([]byte, MaxPayloadSize+1))
	assert.Equal(t, err, ErrPayloadOversized)

	// Create a tx with valid payload
	tx, err = NewMessageTransaction(*from, *to, big.NewInt(100), big.NewInt(0), 38, []byte("hello"))
	assert.Equal(t, err, error(nil))
	tx.Data.Payload = make([]byte, MaxPayloadSize+1) // modify the payload to invalid size.

	statedb := newTestStateDB(tx.Data.From, 38, 200)

	err = tx.Validate(statedb)
	assert.Equal(t, err, ErrPayloadOversized)
}

func prepareShardEnv(localShard uint) func() {
	prevDisabled := common.IsShardDisabled
	prevNum := common.LocalShardNumber

	common.IsShardDisabled = false
	common.LocalShardNumber = localShard

	return func() {
		common.LocalShardNumber = prevNum
		common.IsShardDisabled = prevDisabled
	}
}

func Test_Transaction_Validate_InvalidFromShard(t *testing.T) {
	dispose := prepareShardEnv(9)
	defer dispose()

	from := crypto.MustGenerateShardAddress(1) // invalid shard
	to := crypto.MustGenerateShardAddress(9)
	tx := NewTransaction(*from, *to, big.NewInt(20), big.NewInt(10), 5)

	statedb := newTestStateDB(tx.Data.From, 5, 100)

	err := tx.Validate(statedb)
	assert.Equal(t, strings.Contains(err.Error(), "invalid from address"), true)
}

func Test_Transaction_Validate_InvalidToShard(t *testing.T) {
	dispose := prepareShardEnv(9)
	defer dispose()

	from := crypto.MustGenerateShardAddress(9)
	to := crypto.MustGenerateShardAddress(1) // invalid shard
	tx := NewTransaction(*from, *to, big.NewInt(20), big.NewInt(10), 5)

	statedb := newTestStateDB(tx.Data.From, 5, 100)

	err := tx.Validate(statedb)
	assert.Equal(t, strings.Contains(err.Error(), "invalid to address"), true)
}

func Test_Transaction_Validate_InvalidContractShard(t *testing.T) {
	dispose := prepareShardEnv(9)
	defer dispose()

	// From and contract addresses match the shard number.
	from := crypto.MustGenerateShardAddress(9)
	contractAddr := crypto.MustGenerateShardAddress(9)
	tx, err := NewMessageTransaction(*from, *contractAddr, big.NewInt(20), big.NewInt(10), 5, []byte("contract message"))
	assert.Equal(t, err, error(nil))

	statedb := newTestStateDB(tx.Data.From, 5, 100)

	// Invalid shard of contract creator address
	to := crypto.MustGenerateShardAddress(1)
	statedb.AddContract(*to, *contractAddr)

	err = tx.Validate(statedb)
	assert.Equal(t, strings.Contains(err.Error(), "invalid to address"), true)
}
