/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
	"gopkg.in/fatih/set.v0"
)

func newTestDebt(amount int64, price int64) *types.Debt {
	fromAddress, fromPrivKey := crypto.MustGenerateShardKeyPair(1)
	toAddress := crypto.MustGenerateShardAddress(2)
	tx, _ := types.NewTransaction(*fromAddress, *toAddress, big.NewInt(amount), big.NewInt(price), 1)
	tx.Sign(fromPrivKey)

	return types.NewDebt(tx)
}

func newTestDebtBlock(bc *Blockchain, parentHash common.Hash, blockHeight uint64, num int) *types.Block {
	minerAccount := newTestAccount(consensus.GetReward(blockHeight), 0, 2)
	rewardTx, _ := types.NewRewardTransaction(minerAccount.addr, minerAccount.amount, uint64(1))

	txs := []*types.Transaction{rewardTx}
	var debts []*types.Debt
	for i := 0; i < num; i++ {
		d := newTestDebt(1, 10)
		debts = append(debts, d)
	}

	header := &types.BlockHeader{
		PreviousBlockHash: parentHash,
		Creator:           minerAccount.addr,
		StateHash:         common.EmptyHash,
		TxHash:            types.MerkleRootHash(txs),
		TxDebtHash:        types.DebtMerkleRootHash(types.NewDebts(txs)),
		DebtHash:          types.DebtMerkleRootHash(debts),
		Height:            blockHeight,
		Difficulty:        big.NewInt(1),
		CreateTimestamp:   big.NewInt(1),
		Witness:           make([]byte, 0),
		ExtraData:         make([]byte, 0),
	}

	stateRootHash := common.EmptyHash
	receiptsRootHash := common.EmptyHash
	parentBlock, err := bc.bcStore.GetBlock(parentHash)
	if err == nil {
		statedb, err := state.NewStatedb(parentBlock.Header.StateHash, bc.accountStateDB)
		if err != nil {
			panic(err)
		}

		common.LocalShardNumber = 2
		defer func() {
			common.LocalShardNumber = 0
		}()

		for _, d := range debts {
			err := ApplyDebt(statedb, d, minerAccount.addr, nil)
			if err != nil {
				panic(err)
			}
		}

		var receipts []*types.Receipt
		if receipts, err = bc.updateStateDB(statedb, rewardTx, txs[1:], header); err != nil {
			panic(err)
		}

		if stateRootHash, err = statedb.Hash(); err != nil {
			panic(err)
		}

		receiptsRootHash = types.ReceiptMerkleRootHash(receipts)
	}

	header.StateHash = stateRootHash
	header.ReceiptHash = receiptsRootHash

	return &types.Block{
		HeaderHash:   header.Hash(),
		Header:       header,
		Transactions: txs,
		Debts:        debts,
	}
}

func Test_DebtPool(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)
	pool := NewDebtPool(bc, nil)

	b1 := newTestDebtBlock(bc, bc.genesisBlock.HeaderHash, 1, 2)
	b2 := newTestDebtBlock(bc, bc.genesisBlock.HeaderHash, 1, 2)

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
	reinject := pool.getReinjectDebts(b2.HeaderHash, b1.HeaderHash)
	assert.Equal(t, len(reinject), 2)
	expectedResult := set.New(b1.Debts[0].Hash, b1.Debts[1].Hash)
	assert.Equal(t, expectedResult.Has(reinject[0].Hash), true)
	assert.Equal(t, expectedResult.Has(reinject[1].Hash), true)

	// test remove
	// make b2 be in the block index
	b3 := newTestDebtBlock(bc, b2.HeaderHash, 2, 0)
	bc.WriteBlock(b3)

	common.LocalShardNumber = 2
	defer func() {
		common.LocalShardNumber = common.UndefinedShardNumber
	}()

	pool.add(b1.Debts)
	pool.add(b2.Debts)

	assert.Equal(t, 4, len(pool.hashMap))

	pool.removeDebts()

	assert.Equal(t, len(pool.hashMap), 2)
	assert.Equal(t, pool.hashMap[b1.Debts[0].Hash], b1.Debts[0])
	assert.Equal(t, pool.hashMap[b1.Debts[1].Hash], b1.Debts[1])
}
