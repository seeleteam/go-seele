/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	leveldbErrors "github.com/syndtr/goleveldb/leveldb/errors"
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
	info   *GenesisInfo
}

// GenesisInfo genesis info for generating genesis block, it could be used for initializing account balance
type GenesisInfo struct {
	// Accounts accounts info for genesis block used for test
	// map key is account address -> value is account balance
	Accounts map[common.Address]*big.Int `json:"accounts,omitempty"`

	// Difficult initial difficult for mining. Use bigger difficult as you can. Because block is chosen by total difficult
	Difficult int64 `json:"difficult"`

	// ShardNumber is the shard number of genesis block.
	ShardNumber uint `json:"shard"`

	// CreateTimestamp is the initial time of genesis
	CreateTimestamp *big.Int `json:"timestamp"`

	// Consensus consensus type
	Consensus types.ConsensusType `json:"consensus"`

	// Validators Bft consensus validators
	Validators []common.Address `json:"validators"`

	// master account
	Masteraccount common.Address `json:"master"`

	// balance of the master account
	Balance *big.Int `json:"balance"`
}

// NewGenesisInfo mainchain genesis block info constructor
func NewGenesisInfo(accounts map[common.Address]*big.Int, difficult int64, shard uint, timestamp *big.Int,
	consensus types.ConsensusType, validators []common.Address) *GenesisInfo {

	var masteraccount common.Address
	var balance *big.Int
	if shard == 1 {
		masteraccount, _ = common.HexToAddress("0xd9dd0a837a3eb6f6a605a5929555b36ced68fdd1")
		balance = big.NewInt(17500000000000000)
	} else if shard == 2 {
		masteraccount, _ = common.HexToAddress("0xc71265f11acdacffe270c4f45dceff31747b6ac1")
		balance = big.NewInt(17500000000000000)
	} else if shard == 3 {
		masteraccount, _ = common.HexToAddress("0x509bb3c2285a542e96d3500e1d04f478be12faa1")
		balance = big.NewInt(17500000000000000)
	} else if shard == 4 {
		masteraccount, _ = common.HexToAddress("0xc6c5c85c585ee33aae502b874afe6cbc3727ebf1")
		balance = big.NewInt(17500000000000000)
	} else {
		masteraccount, _ = common.HexToAddress("0x0000000000000000000000000000000000000000")
		balance = big.NewInt(0)
	}
	return &GenesisInfo{
		Accounts:        accounts,
		Difficult:       difficult,
		ShardNumber:     shard,
		CreateTimestamp: timestamp,
		Consensus:       consensus,
		Validators:      validators,
		Masteraccount:   masteraccount,
		Balance:         balance,
	}
}

//NewGenesisInfoSubchain subchain genesis block constructor
func NewGenesisInfoSubchain(accounts map[common.Address]*big.Int, difficult int64, shard uint, timestamp *big.Int,
	consensus types.ConsensusType, validator []common.Address, masteraccount common.Address, balance *big.Int) *GenesisInfo {
	return &GenesisInfo{
		Accounts:        accounts,
		Difficult:       difficult,
		ShardNumber:     shard,
		CreateTimestamp: timestamp,
		Consensus:       consensus,
		Validators:      validator,
		Masteraccount:   masteraccount,
		Balance:         balance,
	}
}

// Hash returns GenesisInfo hash
func (info *GenesisInfo) Hash() common.Hash {
	data, err := json.Marshal(info)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal err: %s", err))
	}

	return crypto.HashBytes(data)
}

// shardInfo represents the extra data that saved in the genesis block in the blockchain.
type shardInfo struct {
	ShardNumber uint
}

// GetGenesis gets the genesis block according to accounts' balance
func GetGenesis(info *GenesisInfo) *Genesis {
	if info.Difficult <= 0 {
		info.Difficult = 1
	}

	statedb := getStateDB(info)
	stateRootHash, err := statedb.Hash()
	if err != nil {
		panic(err)
	}

	extraData := []byte{}
	if info.Consensus == types.IstanbulConsensus || info.Consensus == types.BftConsensus {
		extraData = generateConsensusInfo(info.Validators)
		fmt.Printf("consensus is %d extraData initiated with %v", types.BftConsensus, info.Validators)
	}

	shard := common.SerializePanic(shardInfo{
		ShardNumber: info.ShardNumber,
	})

	fmt.Println("[subchain] the genesis consensus algorithm", info.Consensus)

	return &Genesis{
		header: &types.BlockHeader{
			PreviousBlockHash: common.EmptyHash,
			Creator:           common.EmptyAddress,
			StateHash:         stateRootHash,
			TxHash:            types.MerkleRootHash(nil),
			Difficulty:        big.NewInt(info.Difficult),
			Height:            genesisBlockHeight,
			CreateTimestamp:   info.CreateTimestamp,
			Consensus:         info.Consensus,
			Witness:           shard,
			ExtraData:         extraData,
		},
		info: info,
	}
}

func generateConsensusInfo(addrs []common.Address) []byte {
	var consensusInfo []byte
	consensusInfo = append(consensusInfo, bytes.Repeat([]byte{0x00}, types.IstanbulExtraVanity)...)

	ist := &types.IstanbulExtra{
		Validators:    addrs,
		Seal:          []byte{},
		CommittedSeal: [][]byte{},
	}

	istPayload, err := rlp.EncodeToBytes(&ist)
	if err != nil {
		panic("failed to encode istanbul extra")
	}

	consensusInfo = append(consensusInfo, istPayload...)
	return consensusInfo
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
	if err == leveldbErrors.ErrNotFound {
		return genesis.store(bcStore, accountStateDB)
	}

	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get block hash by height %v in canonical chain", genesisBlockHeight)
	}

	storedGenesis, err := bcStore.GetBlock(storedGenesisHash)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get genesis block by hash %v", storedGenesisHash)
	}

	data, err := getShardInfo(storedGenesis)
	if err != nil {
		return errors.NewStackedError(err, "failed to get extra data in genesis block")
	}

	if data.ShardNumber != genesis.info.ShardNumber {
		return fmt.Errorf("specific shard number %d does not match with the shard number in genesis info %d", data.ShardNumber, genesis.info.ShardNumber)
	}

	if headerHash := genesis.header.Hash(); !headerHash.Equal(storedGenesisHash) {
		return ErrGenesisHashMismatch
	}

	return nil
}

// store atomically stores the genesis block in the blockchain store.
func (genesis *Genesis) store(bcStore store.BlockchainStore, accountStateDB database.Database) error {
	statedb := getStateDB(genesis.info)

	batch := accountStateDB.NewBatch()
	if _, err := statedb.Commit(batch); err != nil {
		return errors.NewStackedError(err, "failed to commit batch into statedb")
	}

	if err := batch.Commit(); err != nil {
		return errors.NewStackedError(err, "failed to commit batch into database")
	}

	if err := bcStore.PutBlockHeader(genesis.header.Hash(), genesis.header, genesis.header.Difficulty, true); err != nil {
		return errors.NewStackedError(err, "failed to put genesis block header into store")
	}

	return nil
}

func getStateDB(info *GenesisInfo) *state.Statedb {
	statedb := state.NewEmptyStatedb(nil)

	if info.ShardNumber == 1 {
		info.Masteraccount, _ = common.HexToAddress("0xd9dd0a837a3eb6f6a605a5929555b36ced68fdd1")
		info.Balance = big.NewInt(17500000000000000)
		statedb.CreateAccount(info.Masteraccount)
		statedb.SetBalance(info.Masteraccount, info.Balance)
	} else if info.ShardNumber == 2 {
		info.Masteraccount, _ = common.HexToAddress("0xc71265f11acdacffe270c4f45dceff31747b6ac1")
		info.Balance = big.NewInt(17500000000000000)
		statedb.CreateAccount(info.Masteraccount)
		statedb.SetBalance(info.Masteraccount, info.Balance)
	} else if info.ShardNumber == 3 {
		info.Masteraccount, _ = common.HexToAddress("0x509bb3c2285a542e96d3500e1d04f478be12faa1")
		info.Balance = big.NewInt(17500000000000000)
		statedb.CreateAccount(info.Masteraccount)
		statedb.SetBalance(info.Masteraccount, info.Balance)
	} else if info.ShardNumber == 4 {
		info.Masteraccount, _ = common.HexToAddress("0xc6c5c85c585ee33aae502b874afe6cbc3727ebf1")
		info.Balance = big.NewInt(17500000000000000)
		statedb.CreateAccount(info.Masteraccount)
		statedb.SetBalance(info.Masteraccount, info.Balance)
	} else {
		info.Masteraccount, _ = common.HexToAddress("0x0000000000000000000000000000000000000000")
		info.Balance = big.NewInt(0)
	}

	for addr, amount := range info.Accounts {
		if !common.IsShardEnabled() || addr.Shard() == info.ShardNumber {
			statedb.CreateAccount(addr)
			statedb.SetBalance(addr, amount)
		}
	}

	return statedb
}

// getShardInfo returns the extra data of specified genesis block.
func getShardInfo(genesisBlock *types.Block) (*shardInfo, error) {
	if genesisBlock.Header.Height != genesisBlockHeight {
		return nil, fmt.Errorf("invalid genesis block height %v", genesisBlock.Header.Height)
	}

	data := &shardInfo{}
	if err := common.Deserialize(genesisBlock.Header.Witness, data); err != nil {
		return nil, errors.NewStackedError(err, "failed to deserialize the extra data of genesis block")
	}

	return data, nil
}
