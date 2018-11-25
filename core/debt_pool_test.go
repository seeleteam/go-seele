/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/stretchr/testify/assert"
	"gopkg.in/fatih/set.v0"
)

func Test_DebtPool(t *testing.T) {
	bc := NewTestBlockchain()
	pool := NewDebtPool(bc, nil)

	b1 := newTestBlockWithDebt(bc, bc.genesisBlock.HeaderHash, 1, 2*types.DebtSize, true)
	b2 := newTestBlockWithDebt(bc, bc.genesisBlock.HeaderHash, 1, 2*types.DebtSize, true)
	assert.Equal(t, 2, len(b1.Debts))
	assert.Equal(t, 2, len(b2.Debts))

	common.LocalShardNumber = 2
	defer func() {
		common.LocalShardNumber = common.UndefinedShardNumber
	}()

	err := bc.WriteBlock(b1)
	if err != nil {
		panic(err)
	}

	err = bc.WriteBlock(b2)
	if err != nil {
		panic(err)
	}

	// Test reinject
	reinject := pool.getReinjectObject(b2.HeaderHash, b1.HeaderHash)
	assert.Equal(t, len(reinject), 2)
	expectedResult := set.New(b1.Debts[0].Hash, b1.Debts[1].Hash)
	assert.Equal(t, expectedResult.Has(reinject[0].GetHash()), true)
	assert.Equal(t, expectedResult.Has(reinject[1].GetHash()), true)

	// test remove
	// make b2 be in the block index
	b3 := newTestBlockWithDebt(bc, b2.HeaderHash, 2, 0, true)
	bc.WriteBlock(b3)

	common.LocalShardNumber = 2
	defer func() {
		common.LocalShardNumber = common.UndefinedShardNumber
	}()
	pool.AddDebtArray(b1.Debts)
	pool.AddDebtArray(b2.Debts)

	assert.Equal(t, 4, pool.GetDebtCount(true, true))

	pool.removeObjects()

	assert.Equal(t, pool.getObjectCount(true, true), 2)
	assert.Equal(t, pool.GetDebtByHash(b1.Debts[0].Hash), b1.Debts[0])
	assert.Equal(t, pool.GetDebtByHash(b1.Debts[1].Hash), b1.Debts[1])
}

func Test_OrderByFee(t *testing.T) {
	bc := NewTestBlockchain()
	pool := NewDebtPool(bc, nil)

	d1 := types.NewTestDebtDetail(1, 10)
	d2 := types.NewTestDebtDetail(2, 11)

	common.LocalShardNumber = 2
	defer func() {
		common.LocalShardNumber = common.UndefinedShardNumber
	}()
	pool.AddDebtArray([]*types.Debt{d1, d2})

	results, _ := pool.GetProcessableDebts(10000)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, results[0].Data.Price.Cmp(results[1].Data.Price), 1)
}

func Test_AddWithValidation(t *testing.T) {
	verifier := types.NewTestVerifier(true, false, nil)
	bc := NewTestBlockchain()
	pool := NewDebtPool(bc, verifier)
	d1 := types.NewTestDebtDetail(1, 10)

	common.LocalShardNumber = 2
	defer func() {
		common.LocalShardNumber = common.UndefinedShardNumber
	}()
	pool.AddDebt(d1)

	assert.Equal(t, 1, pool.GetDebtCount(true, true))
}

func Test_DebtPool_AddBack(t *testing.T) {
	verifier := types.NewTestVerifier(true, false, nil)
	bc := NewTestBlockchain()
	pool := NewDebtPool(bc, verifier)
	d1 := types.NewTestDebtDetail(1, 10)
	d2 := types.NewTestDebtDetail(2, 10)
	d3 := types.NewTestDebtDetail(3, 10)

	common.LocalShardNumber = 2
	defer func() {
		common.LocalShardNumber = common.UndefinedShardNumber
	}()
	pool.AddDebt(d1)
	pool.AddDebt(d2)
	pool.AddDebt(d3)

	assert.Equal(t, 3, pool.GetDebtCount(true, true))

	debts, size := pool.GetProcessableDebts(types.DebtSize * 2)
	assert.True(t, size >= types.DebtSize)
	assert.Equal(t, 2, pool.GetDebtCount(true, false))
	assert.Equal(t, 1, pool.GetDebtCount(false, true))

	assert.Equal(t, 2, len(debts))

	pool.AddBackDebts(debts[0:1])
	assert.Equal(t, 2, pool.GetDebtCount(false, true))
	assert.Equal(t, 1, pool.GetDebtCount(true, false))
}
