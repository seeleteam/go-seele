/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

var (
	// ErrGenesisHashMismatch is returned when genesis block hash mismatch between store and memory.
	ErrGenesisHashMismatch = errors.New("genesis block hash mismatch")

	// ErrGenesisNotFound is returned when genesis block not found in store.
	ErrGenesisNotFound = errors.New("genesis block not found")
)

const genesisBlockHeight = uint64(0)

// Genesis represents the genesis block in the blockchain.
type Genesis struct {
	bcStore  store.BlockchainStore
	header   *types.BlockHeader
	accounts map[common.Address]state.Account
}

// DefaultGenesis returns the default genesis block in the blockchain.
// TODO default genesis value is TBD according to the consensus algorithm.
func DefaultGenesis(bcStore store.BlockchainStore) *Genesis {
	// TODO define default accounts in genesis block
	defaultAccounts := map[common.Address]state.Account{
		common.HexMustToAddres("0x55489251c9d3b394e430d50cb20e271c8560d39b02dfb7efe9610ff51fa4affcf663ad4337117263f64b24149fed5c4fe95d5fb3a00d45a32e6433a200fa0301"): state.Account{0, big.NewInt(10000)},
		common.HexMustToAddres("0x2d7d61c30a2f62cacc84bdd17759da7498ba7f0b9081f501a3a4c37c492eb493a0dcd59caaa7284bf38500d4d896cbb0caea504e5b9b3d1802433d06465a0a23"): state.Account{0, big.NewInt(20000)},
		common.HexMustToAddres("0x3acdcc24c04c893280823715c4046df9d28d1f5ee362ad70e066932ee2c3b836b264d3897d1a9b788884362a75e7da0a89669f6f86ce52f2b73858a8e3f065d8"): state.Account{0, big.NewInt(30000)},
	}

	statedb, err := state.NewStatedb(common.EmptyHash, nil)
	if err != nil {
		panic(err)
	}

	for addr, account := range defaultAccounts {
		stateObj := statedb.GetOrNewStateObject(addr)
		stateObj.SetNonce(account.Nonce)
		stateObj.SetAmount(account.Amount)
	}

	stateRootHash, err := statedb.Commit(nil)
	if err != nil {
		panic(err)
	}

	return &Genesis{
		bcStore: bcStore,
		header: &types.BlockHeader{
			PreviousBlockHash: common.EmptyHash,
			Creator:           common.Address{},
			StateHash:         stateRootHash,
			TxHash:            types.MerkleRootHash(nil),
			Difficulty:        big.NewInt(1),
			Height:            genesisBlockHeight,
			CreateTimestamp:   big.NewInt(0),
			Nonce:             1,
		},
		accounts: defaultAccounts,
	}
}

// Initialize writes the genesis block in blockchain store if unavailable.
// Otherwise, check if the existing genesis block is valid in blockchain store.
func (genesis *Genesis) Initialize(accountStateDB database.Database) error {
	storedGenesisHash, err := genesis.bcStore.GetBlockHash(genesisBlockHeight)

	// FIXME use seele defined common error instead of concrete levelDB error.
	if err == errors.ErrNotFound {
		return genesis.store(accountStateDB)
	}

	if err != nil {
		return err
	}

	headerHash := genesis.header.Hash()
	if !headerHash.Equal(storedGenesisHash) {
		return ErrGenesisHashMismatch
	}

	return nil
}

// store atomically stores the genesis block in blockchain store.
func (genesis *Genesis) store(accountStateDB database.Database) error {
	statedb, err := state.NewStatedb(common.EmptyHash, accountStateDB)
	if err != nil {
		return err
	}

	for addr, account := range genesis.accounts {
		stateObj := statedb.GetOrNewStateObject(addr)
		stateObj.SetNonce(account.Nonce)
		stateObj.SetAmount(account.Amount)
	}

	batch := accountStateDB.NewBatch()

	_, err = statedb.Commit(batch)
	if err != nil {
		batch.Rollback()
		return err
	}

	if err = batch.Commit(); err != nil {
		return err
	}

	return genesis.bcStore.PutBlockHeader(genesis.header.Hash(), genesis.header, genesis.header.Difficulty, true)
}
