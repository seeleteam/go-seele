package core

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func newEventPool() (*EventPool, func()) {
	chain := newMockBlockchain()
	db, dispose := leveldb.NewTestDatabase()
	return NewEventPool(10000, store.NewBlockchainDatabase(db), chain, nil), dispose
}

func getRandomTx() *types.Transaction {
	fromAddress, fromPrivateKey, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(err)
	}

	tx, err := types.NewTransaction(*fromAddress, *crypto.MustGenerateRandomAddress(), big.NewInt(10), big.NewInt(10), 1)
	tx.Sign(fromPrivateKey)

	return tx
}

func newTestFullBlock(debtNum, txNum int) *types.Block {
	var txs []*types.Transaction
	for i := 0; i < txNum; i++ {
		txs = append(txs, getRandomTx())
	}

	var debts []*types.Debt
	for i := 0; i < debtNum; i++ {
		d := types.NewDebtWithContext(getRandomTx())
		debts = append(debts, d)
	}

	header := &types.BlockHeader{
		PreviousBlockHash: crypto.MustHash("a"),
		Creator:           *crypto.MustGenerateRandomAddress(),
		StateHash:         crypto.MustHash("b"),
		TxHash:            types.MerkleRootHash(txs),
		TxDebtHash:        types.DebtMerkleRootHash(types.NewDebts(txs)),
		DebtHash:          types.DebtMerkleRootHash(debts),
		Height:            1,
		Difficulty:        big.NewInt(2),
		CreateTimestamp:   big.NewInt(3),
		Witness:           []byte{0x4},
		ExtraData:         []byte{0x5},
	}

	return &types.Block{
		HeaderHash:   header.Hash(),
		Header:       header,
		Transactions: txs,
		Debts:        debts,
	}
}

func Test_EventPool_getBeginHeight(t *testing.T) {
	pool, dispose := newEventPool()
	defer dispose()

	block := newTestFullBlock(3, 3)
	err := pool.mainChainStore.PutBlock(block, block.Header.Difficulty, true)
	assert.NoError(t, err)

	height, err := pool.getBeginHeight()
	assert.NoError(t, err)
	assert.Equal(t, height, uint64(1))

}
