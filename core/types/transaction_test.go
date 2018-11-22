/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package types

import (
	"crypto/ecdsa"
	"encoding/json"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/params"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/stretchr/testify/assert"
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

func newTestTx(t *testing.T, amount, price, nonce uint64, sign bool) *Transaction {
	fromPrivKey, fromAddress := randomAccount(t)
	toAddress := randomAddress(t)

	tx, err := NewTransaction(fromAddress, toAddress, new(big.Int).SetUint64(amount), new(big.Int).SetUint64(price), nonce)
	if err != nil {
		t.Fatal(err)
	}

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

func newTestStateDB(address common.Address, nonce, balance uint64) *mockStateDB {
	return &mockStateDB{
		balances: map[common.Address]*big.Int{address: new(big.Int).SetUint64(balance)},
		nonces:   map[common.Address]uint64{address: nonce},
	}
}

// Validate successfully if no data changed.
func Test_Transaction_Validate_NoDataChange(t *testing.T) {
	tx := newTestTx(t, 100, 2, 38, true)
	statedb := newTestStateDB(tx.Data.From, 38, 200000)
	err := tx.Validate(statedb)
	assert.Equal(t, err, error(nil))
}

func Benchmark_Transaction_ValidateWithState(b *testing.B) {
	tx := newTestTx(nil, 100, 2, 38, true)
	statedb := newTestStateDB(tx.Data.From, 38, 200)

	for i := 0; i < b.N; i++ {
		tx.Validate(statedb)
	}
}

func Benchmark_Transaction_ValidateWithoutState(b *testing.B) {
	tx := newTestTx(nil, 100, 2, 38, true)

	for i := 0; i < b.N; i++ {
		tx.ValidateWithoutState(true, true)
	}
}

func Benchmark_Transaction_ValidateWithoutSig(b *testing.B) {
	tx := newTestTx(nil, 100, 2, 38, true)

	for i := 0; i < b.N; i++ {
		tx.ValidateWithoutState(false, true)
	}
}

func Benchmark_Transaction_ParallelValidate(b *testing.B) {
	tx := newTestTx(nil, 100, 2, 38, true)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tx.ValidateWithoutState(true, true)
		}
	})
}

func Benchmark_Transaction_Sign(b *testing.B) {
	fromPrivKey, fromAddress := randomAccount(nil)
	toAddress := randomAddress(nil)
	tx, _ := NewTransaction(fromAddress, toAddress, big.NewInt(1), big.NewInt(2), 1)

	for i := 0; i < b.N; i++ {
		tx.Sign(fromPrivKey)
	}
}

func Benchmark_Transaction_MerkleRootHash(b *testing.B) {
	txs := []*Transaction{
		newTestTx(nil, 1, 2, 3, true),
		newTestTx(nil, 4, 5, 6, true),
		newTestTx(nil, 7, 8, 9, true),
	}

	for i := 0; i < b.N; i++ {
		MerkleRootHash(txs)
	}
}

// Validate failed if transaction not signed.
func Test_Transaction_Validate_NotSigned(t *testing.T) {
	tx := newTestTx(t, 100, 2, 38, false)
	statedb := newTestStateDB(tx.Data.From, 38, 200)
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrSigMissing)
}

// Validate failed if transaction Hash value changed.
func Test_Transaction_Validate_HashChanged(t *testing.T) {
	tx := newTestTx(t, 100, 2, 38, true)
	tx.Hash = crypto.HashBytes([]byte("test"))
	statedb := newTestStateDB(tx.Data.From, 38, 200)
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrHashMismatch)
}

// Validate failed if transaction data changed.
func Test_Transaction_Validate_TxDataChanged(t *testing.T) {
	tx := newTestTx(t, 100, 2, 38, true)
	tx.Data.Amount.SetInt64(200)
	statedb := newTestStateDB(tx.Data.From, 38, 200)
	err := tx.Validate(statedb)
	assert.Equal(t, err, ErrHashMismatch)
}

// Validate failed if transaction data changed along with Hash updated.
func Test_Transaction_Validate_SignInvalid(t *testing.T) {
	tx := newTestTx(t, 100, 2, 38, true)

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
	tx := newTestTx(t, 100, 2, 38, true)
	statedb := newTestStateDB(tx.Data.From, 38, 101)
	err := tx.Validate(statedb)
	assert.Equal(t, err != nil, true)
}

func Test_Transaction_Validate_NonceTooLow(t *testing.T) {
	tx := newTestTx(t, 100, 2, 38, true)
	statedb := newTestStateDB(tx.Data.From, 40, 200)
	err := tx.Validate(statedb)
	assert.Equal(t, err != nil, true)
}

func Test_Transaction_Validate_PayloadOversized(t *testing.T) {
	from := crypto.MustGenerateRandomAddress()
	to := crypto.MustGenerateRandomAddress()

	// Cannot create a tx with oversized payload.
	tx, err := NewMessageTransaction(*from, *to, big.NewInt(100), big.NewInt(1), math.MaxUint64, 38, make([]byte, MaxPayloadSize+1))
	assert.Equal(t, err, ErrPayloadOversized)

	// Create a tx with valid payload
	tx, err = NewMessageTransaction(*from, *to, big.NewInt(100), big.NewInt(1), math.MaxUint64, 38, []byte("hello"))
	assert.Equal(t, err, error(nil))
	tx.Data.Payload = make([]byte, MaxPayloadSize+1) // modify the payload to invalid size.

	statedb := newTestStateDB(tx.Data.From, 38, 200)

	err = tx.Validate(statedb)
	assert.Equal(t, err, ErrPayloadOversized)
}

func Test_Transaction_Validate_PayLoadJSON(t *testing.T) {
	tx := newTestTx(t, 100, 2, 38, true)
	assert.Equal(t, len(tx.Data.Payload), 0)

	arrayByte, err := json.Marshal(tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, strings.Contains(string(arrayByte), "Payload\":\"0x"), false)

	tx1 := Transaction{}
	err = json.Unmarshal(arrayByte, &tx1)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(tx1.Data.Payload), 0)

	from := crypto.MustGenerateRandomAddress()
	to := crypto.MustGenerateRandomAddress()
	tx2, err := NewMessageTransaction(*from, *to, big.NewInt(100), big.NewInt(1), math.MaxUint64, 38, []byte("hello"))
	assert.Equal(t, err, nil)
	assert.Equal(t, string(tx2.Data.Payload), "hello")

	arrayByte1, err := json.Marshal(tx2)
	assert.Equal(t, err, nil)
	assert.Equal(t, strings.Contains(string(arrayByte1), "Payload\":\"0x"), true)

	tx3 := Transaction{}
	err = json.Unmarshal(arrayByte1, &tx3)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx3.Data.Payload, tx2.Data.Payload)
}

func prepareShardEnv(localShard uint) func() {
	prevNum := common.LocalShardNumber
	common.LocalShardNumber = localShard

	return func() {
		common.LocalShardNumber = prevNum
	}
}

func Test_Transaction_InvalidPrice(t *testing.T) {
	dispose := prepareShardEnv(2)
	defer dispose()

	// From and contract addresses match the shard number.
	from := crypto.MustGenerateShardAddress(2)
	contractAddr := crypto.MustGenerateShardAddress(2)

	tx, err := NewTransaction(*from, *contractAddr, big.NewInt(20), big.NewInt(-1), 5)
	assert.Equal(t, tx, (*Transaction)(nil))
	assert.Equal(t, err, ErrPriceNegative)

	tx, err = NewTransaction(*from, *contractAddr, big.NewInt(20), nil, 5)
	assert.Equal(t, tx, (*Transaction)(nil))
	assert.Equal(t, err, ErrPriceNil)
}

func Test_Transaction_EmptyPayloadError(t *testing.T) {
	from := *crypto.MustGenerateRandomAddress()

	_, err := NewContractTransaction(from, big.NewInt(100), big.NewInt(2), math.MaxUint64, 38, nil)
	assert.Equal(t, err, ErrPayloadEmpty)

	contractAddr := crypto.CreateAddress(from, 77)
	_, err = NewMessageTransaction(from, contractAddr, big.NewInt(100), big.NewInt(2), math.MaxUint64, 38, nil)
	assert.Equal(t, err, ErrPayloadEmpty)
}

func Test_Transaction_Validate_EmptyPayloadError(t *testing.T) {
	fromPrivKey, fromAddr := randomAccount(t)
	toAddress := crypto.CreateAddress(fromAddr, 38)

	tx, err := newTx(fromAddr, toAddress, big.NewInt(100), big.NewInt(2), math.MaxUint64, 38, []byte("payload"))
	assert.Equal(t, err, nil)

	tx.Data.Payload = nil
	tx.Sign(fromPrivKey)

	statedb := newTestStateDB(tx.Data.From, 38, 200)
	assert.Equal(t, tx.Validate(statedb), ErrPayloadEmpty)
}

func assertTxRlp(t *testing.T, tx *Transaction) {
	encoded := common.SerializePanic(tx)

	tx2 := &Transaction{}
	assert.Equal(t, common.Deserialize(encoded, tx2), nil)
	assert.Equal(t, tx, tx2)
}

func Test_Transaction_RlpTransferTx(t *testing.T) {
	from := *crypto.MustGenerateRandomAddress()
	to := *crypto.MustGenerateRandomAddress()
	tx, err := NewTransaction(from, to, big.NewInt(3), big.NewInt(1), 38)
	assert.Equal(t, err, nil)

	assertTxRlp(t, tx)
}

func Test_Transaction_RlpContractTx(t *testing.T) {
	from := *crypto.MustGenerateRandomAddress()
	tx, err := NewContractTransaction(from, big.NewInt(3), big.NewInt(1), math.MaxUint64, 38, []byte("test code"))
	assert.Equal(t, err, nil)

	assertTxRlp(t, tx)
}

func Test_Transaction_RlpMsgTx(t *testing.T) {
	from := *crypto.MustGenerateRandomAddress()
	to := *crypto.MustGenerateRandomAddress()
	contractAddr := crypto.CreateAddress(to, 38)
	tx, err := NewMessageTransaction(from, contractAddr, big.NewInt(3), big.NewInt(1), math.MaxUint64, 38, []byte("test input message"))
	assert.Equal(t, err, nil)

	assertTxRlp(t, tx)
}

func Test_Transaction_RlpRewardTx(t *testing.T) {
	miner := *crypto.MustGenerateRandomAddress()
	tx, err := NewRewardTransaction(miner, big.NewInt(10), 666)
	assert.Equal(t, err, nil)

	assertTxRlp(t, tx)
}

func Test_Transaction_InvalidAmount(t *testing.T) {
	_, fromAddress := randomAccount(t)
	toAddress := randomAddress(t)

	_, err := NewTransaction(fromAddress, toAddress, nil, new(big.Int).SetInt64(1), 0)
	assert.Equal(t, err, ErrAmountNil)

	_, err = NewTransaction(fromAddress, toAddress, big.NewInt(-1), new(big.Int).SetInt64(1), 0)
	assert.Equal(t, err, ErrAmountNegative)
}

func Test_Transaction_IntrinsicGasError(t *testing.T) {
	from := *crypto.MustGenerateRandomAddress()
	to := *crypto.MustGenerateRandomAddress()
	tx, err := newTx(from, to, big.NewInt(38), big.NewInt(1), 1, 1, nil)

	assert.Nil(t, tx)
	assert.Equal(t, ErrIntrinsicGas, err)
}

func Test_Transaction_IntrinsicGasOverflow(t *testing.T) {
	overflowPayloadSize := (math.MaxUint64 - params.TxGas) / params.TxDataNonZeroGas
	assert.Equal(t, overflowPayloadSize > defaultMaxPayloadSize, true)
}

func Test_Transaction_BatchValidateTxs_NoSig(t *testing.T) {
	var txs []*Transaction

	for i := 0; i < 100; i++ {
		txs = append(txs, newTestTx(t, 1, 1, uint64(i), false))
	}

	assert.Equal(t, ErrSigMissing, BatchValidateTxs(txs))
}

func Test_Transaction_SigCache(t *testing.T) {
	sigCache.Purge()

	// succeed to verify signature of valid tx
	tx := newTestTx(t, 1, 1, 1, true)
	assert.NoError(t, tx.verifySignature())
	assert.Equal(t, 1, sigCache.Len())

	// verify again, and cache is used.
	assert.NoError(t, tx.verifySignature())
	assert.Equal(t, 1, sigCache.Len())

	// change the tx signature, and tx hash not changed yet.
	tx.Signature.Sig = []byte{1, 2, 3}
	assert.Equal(t, ErrSigInvalid, tx.verifySignature())
	assert.Equal(t, 2, sigCache.Len())
}
