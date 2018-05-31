package cmd

import (
	"math/big"
	"os"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/core/vm"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
)

// preprocessContract creates the contract tx dependent state DB, blockchain store
func preprocessContract() (*state.Statedb, store.BlockchainStore, func()) {
	db, err := leveldb.NewLevelDB(dir)
	if err != nil {
		os.RemoveAll(dir)
		panic(err)
	}

	hash := common.EmptyHash
	str, err := db.GetString(STATEDBHASH)
	if err == nil {
		if h, err := common.HexToHash(str); err == nil {
			hash = h
		}
	}

	statedb, err := state.NewStatedb(hash, db)
	if err != nil {
		db.Close()
		panic(err)
	}

	return statedb, store.NewBlockchainDatabase(db), func() {
		batch := db.NewBatch()
		hash, err := statedb.Commit(batch)
		if err != nil {
			panic(err)
		}

		if err := batch.Commit(); err != nil {
			panic(err)
		}

		db.PutString(STATEDBHASH, hash.ToHex())
		db.Close()

	}
}

// Create the contract or call the contract
func processContract(statedb *state.Statedb, bcStore store.BlockchainStore, tx *types.Transaction) *types.Receipt {
	header := newBlockHeader()
	evmContext := core.NewEVMContext(tx, header, header.Creator, bcStore)

	receipt, err := core.ProcessContract(evmContext, tx, 0, statedb, &vm.Config{})
	if err != nil {
		panic(err)
	}
	return receipt
}

// A test BlockHeader
func newBlockHeader() *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: crypto.MustHash("block previous hash"),
		Creator:           *crypto.MustGenerateRandomAddress(),
		StateHash:         crypto.MustHash("state root hash"),
		TxHash:            crypto.MustHash("tx root hash"),
		ReceiptHash:       crypto.MustHash("receipt root hash"),
		Difficulty:        big.NewInt(38),
		Height:            666,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		Nonce:             NONCE,
		ExtraData:         make([]byte, 0),
	}
}
