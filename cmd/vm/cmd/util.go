/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/svm"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
)

// const
const (
	DefaultNonce             = uint64(1)
	KeyStateRootHash         = "STATE_ROOT_HASH"
	keyGlobalContractAddress = "GLOBAL_CONTRACT_ADDRESS"
)

var prefixFuncHash = []byte("FH-")

func getGlobalContractAddress(db database.Database) common.Address {
	hexAddr, err := db.GetString(keyGlobalContractAddress)
	if err != nil {
		return common.EmptyAddress
	}

	return common.HexMustToAddres(hexAddr)
}

func setGlobalContractAddress(db database.Database, hexAddr string) {
	if err := db.PutString(keyGlobalContractAddress, hexAddr); err != nil {
		panic(err)
	}
}

func setContractCompilationOutput(db database.Database, contractAddress []byte, output *solCompileOutput) {
	key := append(prefixFuncHash, contractAddress...)

	if err := db.Put(key, common.SerializePanic(output)); err != nil {
		panic(err)
	}
}

func getContractCompilationOutput(db database.Database, contractAddress []byte) *solCompileOutput {
	key := append(prefixFuncHash, contractAddress...)

	value, err := db.Get(key)
	if err != nil {
		return nil
	}

	output := solCompileOutput{}
	if err = common.Deserialize(value, &output); err != nil {
		panic(err)
	}

	return &output
}

func getFromAddress(statedb *state.Statedb) common.Address {
	if len(account) == 0 {
		from := *crypto.MustGenerateRandomAddress()
		statedb.CreateAccount(from)
		statedb.SetBalance(from, common.SeeleToFan)
		statedb.SetNonce(from, DefaultNonce)
		return from
	}

	from, err := common.HexToAddress(account)
	if err != nil {
		fmt.Println("Invalid account address,", err.Error())
		return common.EmptyAddress
	}

	return from
}

func ensurePrefix(str, prefix string) string {
	if strings.HasPrefix(str, prefix) {
		return str
	}

	return prefix + str
}

// preprocessContract creates the contract tx dependent state DB, blockchain store
func preprocessContract() (database.Database, *state.Statedb, store.BlockchainStore, func(), error) {
	db, err := leveldb.NewLevelDB(defaultDir)
	if err != nil {
		os.RemoveAll(defaultDir)
		return nil, nil, nil, func() {}, err
	}

	hash := common.EmptyHash
	str, err := db.GetString(KeyStateRootHash)
	if err != nil {
		hash = common.EmptyHash
	} else {
		h, err := common.HexToHash(str)
		if err != nil {
			db.Close()
			return nil, nil, nil, func() {}, err
		}
		hash = h
	}

	statedb, err := state.NewStatedb(hash, db)
	if err != nil {
		db.Close()
		return nil, nil, nil, func() {}, err
	}

	return db, statedb, store.NewBlockchainDatabase(db), func() {
		batch := db.NewBatch()
		hash, err := statedb.Commit(batch)
		if err != nil {
			fmt.Println("Failed to commit state DB,", err.Error())
			return
		}

		if err := batch.Commit(); err != nil {
			fmt.Println("Failed to commit batch,", err.Error())
			return
		}

		db.PutString(KeyStateRootHash, hash.Hex())
		db.Close()
	}, nil
}

// Create the contract or call the contract
func processContract(statedb *state.Statedb, bcStore store.BlockchainStore, tx *types.Transaction) (*types.Receipt, error) {
	// A test block header
	header := &types.BlockHeader{
		PreviousBlockHash: crypto.MustHash("block previous hash"),
		Creator:           *crypto.MustGenerateRandomAddress(),
		StateHash:         crypto.MustHash("state root hash"),
		TxHash:            crypto.MustHash("tx root hash"),
		ReceiptHash:       crypto.MustHash("receipt root hash"),
		Difficulty:        big.NewInt(38),
		Height:            666,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		ExtraData:         make([]byte, 0),
	}

	ctx := &svm.Context{
		Tx:          tx,
		Statedb:     statedb,
		BlockHeader: header,
		BcStore:     bcStore,
	}
	return svm.Process(ctx)
}
