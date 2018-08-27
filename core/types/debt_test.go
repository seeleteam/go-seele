/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"math/big"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

func Test_NewDebt(t *testing.T) {
	tx1 := newTestTx(t, 1, 1, 1, true)

	d1 := NewDebt(tx1)
	assert.Equal(t, d1.Data.Amount, big.NewInt(1))
	assert.Equal(t, d1.Data.Account, tx1.Data.To)
	assert.Equal(t, d1.Data.Shard, tx1.Data.To.Shard())
	assert.Equal(t, d1.Data.TxHash, tx1.Hash)
	assert.Equal(t, d1.Hash, crypto.MustHash(d1.Data))
}

func Test_MerkleRoot(t *testing.T) {
	debts := make([]*Debt, 0)

	for i := 0; i < 100; i++ {
		tx := newTestTx(t, 1, 1, 1, true)
		d := NewDebt(tx)

		debts = append(debts, d)
	}

	common.LocalShardNumber = 1
	defer func() {
		common.LocalShardNumber = common.UndefinedShardNumber
	}()

	hash := DebtMerkleRootHash(debts)
	if hash == common.EmptyHash {
		t.Fatal("got empty hash")
	}
}

func Test_DebtSize(t *testing.T) {
	tx := newTestTx(t, 1, 1, 1, true)

	d := NewDebt(tx)

	array := []*Debt{d, d}
	buff := common.SerializePanic(array)
	assert.Equal(t, len(buff)/2, DebtSize)
}
