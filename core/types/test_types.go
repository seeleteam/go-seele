/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package types

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

const TestGenesisShard = 1
const TestDebtTargetShard = 2

// genesis account with enough balance (100K seele) for benchmark test
var TestGenesisAccount = NewTestAccount(new(big.Int).Mul(big.NewInt(100000), common.SeeleToFan), 0, TestGenesisShard)

type testAccount struct {
	Addr    common.Address
	PrivKey *ecdsa.PrivateKey
	Amount  *big.Int
	Nonce   uint64
}

func NewTestAccount(amount *big.Int, nonce uint64, shard uint) *testAccount {
	addr, privKey := crypto.MustGenerateShardKeyPair(shard)

	return &testAccount{
		Addr:    *addr,
		PrivKey: privKey,
		Amount:  new(big.Int).Set(amount),
		Nonce:   nonce,
	}
}

func NewTestTransaction() *Transaction {
	return NewTestTxDetail(1, 1, 0)
}

func NewTestCrossShardTransaction() *Transaction {
	return newTestTxWithShard(1, 1, 0, TestDebtTargetShard, true)
}

func NewTestTransactionWithNonce(nonce uint64) *Transaction {
	return NewTestTxDetail(1, 1, nonce)
}

func NewTestCrossShardTransactionWithNonce(nonce uint64) *Transaction {
	return newTestTxWithShard(1, 1, nonce, TestDebtTargetShard, true)
}

func NewTestDebtWithTargetShard(targeShard uint) *Debt {
	return NewTestDebtDetail(1, 1, targeShard)
}

func NewTestDebt() *Debt {
	return NewTestDebtDetail(1, 1, TestDebtTargetShard)
}

func NewTestDebtDetail(amount int64, price int64, targetShard uint) *Debt {
	fromAddress, fromPrivKey := crypto.MustGenerateKeyPairNotShard(targetShard)
	toAddress := crypto.MustGenerateShardAddress(targetShard)
	tx, _ := NewTransaction(*fromAddress, *toAddress, big.NewInt(amount), big.NewInt(price), 1)
	tx.Sign(fromPrivKey)

	return NewDebtWithoutContext(tx)
}

func NewTestTxDetail(amount, price, nonce uint64) *Transaction {
	return newTestTxWithShard(amount, price, nonce, TestGenesisShard, true)
}

func newTestTxWithShard(amount, price, nonce uint64, shard uint, sign bool) *Transaction {
	toAddress := crypto.MustGenerateShardAddress(shard)

	tx, _ := NewTransaction(TestGenesisAccount.Addr, *toAddress, new(big.Int).SetUint64(amount), new(big.Int).SetUint64(price), nonce)

	if sign {
		tx.Sign(TestGenesisAccount.PrivKey)
	}

	return tx
}

func newTestTxWithSign(amount, price, nonce uint64, sign bool) *Transaction {
	return newTestTxWithShard(amount, price, nonce, TestGenesisShard, sign)
}
