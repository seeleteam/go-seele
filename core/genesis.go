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
	header   *types.BlockHeader
	accounts map[common.Address]*big.Int
}

// DefaultGenesis returns the default genesis block in the blockchain.
func GetGenesis(accounts map[common.Address]*big.Int) *Genesis {
	statedb, err := getStateDB(accounts)
	if err != nil {
		panic(err)
	}

	stateRootHash := statedb.Commit(nil)
	return &Genesis{
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
		accounts: accounts,
	}
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
	statedb, err := getStateDB(genesis.accounts)
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

	if accounts != nil {
		for addr, amount := range accounts {
			stateObj := statedb.GetOrNewStateObject(addr)
			stateObj.SetNonce(0)
			stateObj.SetAmount(amount)
		}
	}

	return statedb, nil
}
