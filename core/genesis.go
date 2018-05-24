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
	// ErrGenesisHashMismatch is returned when the genesis block hash between the store and memory mismatch.
	ErrGenesisHashMismatch = errors.New("genesis block hash mismatch")

	// ErrGenesisNotFound is returned when genesis block not found in the store.
	ErrGenesisNotFound = errors.New("genesis block not found")
)

const genesisBlockHeight = uint64(0)

// Genesis represents the genesis block in the blockchain.
type Genesis struct {
	header *types.BlockHeader
	info   GenesisInfo
}

// GenesisInfo genesis info for generating genesis block, it could be used for initializing account balance
type GenesisInfo struct {
	// Accounts accounts info for genesis block used for test
	// map key is account address -> value is account balance
	Accounts map[common.Address]*big.Int `json:"accounts"`

	// Difficult initial difficult for mining. Use bigger difficult as you can. Because block is choose by total difficult
	Difficult int64 `json:"difficult"`
}

// GetGenesis gets the genesis block according to accounts' balance
func GetGenesis(info GenesisInfo) *Genesis {
	if info.Difficult == 0 {
		info.Difficult = 1
	}

	statedb, err := getStateDB(info.Accounts)
	if err != nil {
		panic(err)
	}

	stateRootHash, err := statedb.Commit(nil)
	if err != nil {
		panic(err)
	}

	return &Genesis{
		header: &types.BlockHeader{
			PreviousBlockHash: common.EmptyHash,
			Creator:           common.Address{},
			StateHash:         stateRootHash,
			TxHash:            types.MerkleRootHash(nil),
			Difficulty:        big.NewInt(info.Difficult),
			Height:            genesisBlockHeight,
			CreateTimestamp:   big.NewInt(0),
			Nonce:             1,
		},
		info: info,
	}
}

// GetDefaultGenesis get the default genesis block.
// This is for test only.
func GetDefaultGenesis(accounts map[common.Address]*big.Int) *Genesis {
	info := GenesisInfo{
		Accounts:  accounts,
		Difficult: 1,
	}

	return GetGenesis(info)
}

// InitializeAndValidate writes the genesis block in the blockchain store if unavailable.
// Otherwise, check if the existing genesis block is valid in the blockchain store.
func (genesis *Genesis) InitializeAndValidate(bcStore store.BlockchainStore, accountStateDB database.Database) error {
	storedGenesisHash, err := bcStore.GetBlockHash(genesisBlockHeight)

	// FIXME use seele-defined common error instead of concrete levelDB error.
	if err == errors.ErrNotFound {
		return genesis.store(bcStore, accountStateDB)
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

// store atomically stores the genesis block in the blockchain store.
func (genesis *Genesis) store(bcStore store.BlockchainStore, accountStateDB database.Database) error {
	statedb, err := getStateDB(genesis.info.Accounts)
	if err != nil {
		return err
	}

	batch := accountStateDB.NewBatch()
	statedb.Commit(batch)
	if err = batch.Commit(); err != nil {
		return err
	}

	return bcStore.PutBlockHeader(genesis.header.Hash(), genesis.header, genesis.header.Difficulty, true)
}

func getStateDB(accounts map[common.Address]*big.Int) (*state.Statedb, error) {
	statedb, err := state.NewStatedb(common.EmptyHash, nil)
	if err != nil {
		return nil, err
	}

	for addr, amount := range accounts {
		stateObj := statedb.GetOrNewStateObject(addr)
		stateObj.SetNonce(0)
		stateObj.SetAmount(amount)
	}

	return statedb, nil
}
