/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"fmt"
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

	// Difficult initial difficult for mining. Use bigger difficult as you can. Because block is chosen by total difficult
	Difficult int64 `json:"difficult"`

	// ShardNumber is the shard number of genesis block.
	ShardNumber uint `json:"shard"`
}

// genesisExtraData represents the extra data that saved in the genesis block in the blockchain.
type genesisExtraData struct {
	ShardNumber uint
}

// GetGenesis gets the genesis block according to accounts' balance
func GetGenesis(info GenesisInfo) *Genesis {
	if info.Difficult == 0 {
		info.Difficult = 1
	}

	statedb, err := getStateDB(info)
	if err != nil {
		panic(err)
	}

	stateRootHash, err := statedb.Hash()
	if err != nil {
		panic(err)
	}

	extraData := genesisExtraData{info.ShardNumber}

	return &Genesis{
		header: &types.BlockHeader{
			PreviousBlockHash: common.EmptyHash,
			Creator:           common.EmptyAddress,
			StateHash:         stateRootHash,
			TxHash:            types.MerkleRootHash(nil),
			Difficulty:        big.NewInt(info.Difficult),
			Height:            genesisBlockHeight,
			CreateTimestamp:   big.NewInt(0),
			Nonce:             1,
			ExtraData:         common.SerializePanic(extraData),
		},
		info: info,
	}
}

// GetShardNumber gets the shard number of genesis
func (genesis *Genesis) GetShardNumber() uint {
	return genesis.info.ShardNumber
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

	storedGenesis, err := bcStore.GetBlock(storedGenesisHash)
	if err != nil {
		return fmt.Errorf("failed to get genesis block. %s", err)
	}

	data, err := getGenesisExtraData(storedGenesis)
	if err != nil {
		return fmt.Errorf("failed to get genesis extra data. %s", err)
	}

	if data.ShardNumber != genesis.info.ShardNumber {
		return errors.New("specific shard number does not match with the shard number in genesis info")
	}

	headerHash := genesis.header.Hash()
	if !headerHash.Equal(storedGenesisHash) {
		return ErrGenesisHashMismatch
	}

	return nil
}

// store atomically stores the genesis block in the blockchain store.
func (genesis *Genesis) store(bcStore store.BlockchainStore, accountStateDB database.Database) error {
	statedb, err := getStateDB(genesis.info)
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

func getStateDB(info GenesisInfo) (*state.Statedb, error) {
	statedb, err := state.NewStatedb(common.EmptyHash, nil)
	if err != nil {
		return nil, err
	}

	for addr, amount := range info.Accounts {
		if !common.IsShardEnabled() || addr.Shard() == info.ShardNumber {
			statedb.CreateAccount(addr)
			statedb.SetBalance(addr, amount)
		}
	}

	return statedb, nil
}

// getGenesisExtraData returns the extra data of specified genesis block.
func getGenesisExtraData(genesisBlock *types.Block) (*genesisExtraData, error) {
	if genesisBlock.Header.Height != genesisBlockHeight {
		return nil, fmt.Errorf("invalid genesis block height %v", genesisBlock.Header.Height)
	}

	data := genesisExtraData{}
	if err := common.Deserialize(genesisBlock.Header.ExtraData, &data); err != nil {
		return nil, err
	}

	return &data, nil
}
