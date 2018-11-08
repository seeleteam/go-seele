/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/consensus/pow"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func newTestRecoveryPointFile() (string, func()) {
	dir, err := ioutil.TempDir("", "SeeleCoreRecoveryPoint")
	if err != nil {
		panic(err)
	}

	return filepath.Join(dir, "rp.bin"), func() {
		os.RemoveAll(dir)
	}
}

func newTestRecoverableBlockchain(bcStore store.BlockchainStore, stateDB database.Database, rpFile string) *Blockchain {
	genesis := newTestGenesis()
	if err := genesis.InitializeAndValidate(bcStore, stateDB); err != nil {
		panic(err)
	}

	bc, err := NewBlockchain(bcStore, stateDB, rpFile, pow.NewEngine(1), nil)
	if err != nil {
		panic(err)
	}

	return bc
}

func Test_RecoveryPoint_FileNotSet(t *testing.T) {
	rp, err := loadRecoveryPoint("")
	assert.Equal(t, err, nil)
	assert.Equal(t, *rp, recoveryPoint{})
}

func Test_RecoveryPoint_FileSet(t *testing.T) {
	rpFile, dispose := newTestRecoveryPointFile()
	defer dispose()

	rp, err := loadRecoveryPoint(rpFile)
	assert.Equal(t, err, nil)
	assert.Equal(t, *rp, recoveryPoint{file: rpFile})
}

func Test_RecoveryPoint_Serialization(t *testing.T) {
	rpFile, dispose := newTestRecoveryPointFile()
	defer dispose()
	rp, _ := loadRecoveryPoint(rpFile)

	// before put block
	rp.WritingBlockHash = common.StringToHash("new block hash")
	rp.WritingBlockHeight = 5
	rp.PreviousHeadBlockHash = common.StringToHash("old HEAD block hash")
	rp.PreviousCanonicalBlockHash = common.StringToHash("old canonical block hash")
	rp.LargerHeight = 6
	rp.StaleHash = common.StringToHash("stale block hash")
	rp.serialize()

	rp2, _ := loadRecoveryPoint(rpFile)
	assert.Equal(t, *rp, *rp2)

	// after put block
	rp.onPutBlockEnd()
	rp2, _ = loadRecoveryPoint(rpFile)
	assert.Equal(t, *rp2, recoveryPoint{LargerHeight: 6, StaleHash: common.StringToHash("stale block hash"), file: rpFile})

	//delete larger height blocks
	rp.onDeleteLargerHeightBlocks(9)
	rp2, _ = loadRecoveryPoint(rpFile)
	assert.Equal(t, *rp2, recoveryPoint{LargerHeight: 9, StaleHash: common.StringToHash("stale block hash"), file: rpFile})

	// overwrite stale blocks in canonical chain
	rp.onDeleteLargerHeightBlocks(0)
	rp.onOverwriteStaleBlocks(common.StringToHash("stale block hash 2"))
	rp2, _ = loadRecoveryPoint(rpFile)
	assert.Equal(t, *rp2, recoveryPoint{StaleHash: common.StringToHash("stale block hash 2"), file: rpFile})
}

func Test_RecoveryPoint_PutBlockCorrupted(t *testing.T) {
	rpFile, dispose1 := newTestRecoveryPointFile()
	defer dispose1()

	db, dispose2 := leveldb.NewTestDatabase()
	defer dispose2()

	// mock corrupt when put a block in DB.
	bcStore := store.NewMemStore()
	bcStore.CorruptOnPutBlock = true

	// should fail to write block due to DB corruption
	// and the inserted block exists in DB
	bc := newTestRecoverableBlockchain(bcStore, db, rpFile)
	newBlock := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	assert.True(t, errors.IsOrContains(bc.WriteBlock(newBlock), store.ErrDBCorrupt))

	// the inserted block exists in DB after corruption
	_, err := bcStore.GetBlock(newBlock.HeaderHash)
	assert.Equal(t, err, nil)

	// the previous inserted block should not exist in DB anymore after recover
	newTestRecoverableBlockchain(bcStore, db, rpFile)
	if _, err = bcStore.GetBlock(newBlock.HeaderHash); err == nil {
		t.Fatal()
	}
}

func Test_RecoveryPoint_RecoverDeleteLargerHeightBlocks(t *testing.T) {
	// height 7 block not deleted before corruption
	rp := recoveryPoint{LargerHeight: 7}
	bcStore := store.NewMemStore()
	block7 := newTestRPBlock(common.StringToHash("block 7"), 7)
	bcStore.PutBlock(block7, big.NewInt(7), true)
	block8 := newTestRPBlock(common.StringToHash("block 8"), 8)
	bcStore.PutBlock(block8, big.NewInt(8), true)

	assert.Equal(t, rp.recover(bcStore), nil)

	if _, err := bcStore.GetBlockHash(7); err == nil {
		t.Fatal()
	}

	if _, err := bcStore.GetBlockHash(8); err == nil {
		t.Fatal()
	}

	// height 7 block already deleted before corruption
	rp = recoveryPoint{LargerHeight: 7}
	bcStore = store.NewMemStore()
	bcStore.PutBlock(block8, big.NewInt(8), true)

	assert.Equal(t, rp.recover(bcStore), nil)

	if _, err := bcStore.GetBlockHash(8); err == nil {
		t.Fatal()
	}
}

func newTestRPBlock(preBlockHash common.Hash, height uint64) *types.Block {
	header := &types.BlockHeader{
		PreviousBlockHash: preBlockHash,
		Height:            height,
	}

	return &types.Block{
		Header:     header,
		HeaderHash: header.Hash(),
	}
}

func Test_RecoveryPoint_RecoverOverwriteStaleBlocks(t *testing.T) {
	bcStore := store.NewMemStore()

	block3 := newTestRPBlock(common.StringToHash("block 2"), 3)
	bcStore.PutBlock(block3, big.NewInt(3), true)

	// old canonical chain
	block41 := newTestRPBlock(block3.HeaderHash, 4)
	bcStore.PutBlock(block41, big.NewInt(4), true)
	block51 := newTestRPBlock(block41.HeaderHash, 5)
	bcStore.PutBlock(block51, big.NewInt(5), true)

	// new canonical chain
	block42 := newTestRPBlock(block3.HeaderHash, 4)
	bcStore.PutBlock(block42, big.NewInt(4), false)
	block52 := newTestRPBlock(block42.HeaderHash, 5)
	bcStore.PutBlock(block52, big.NewInt(5), false)

	// recover: overwrite stale blocks from block52
	// the common ancester is block3, so height 4 and 5
	// in canonical chain will be overwritten.
	rp := recoveryPoint{StaleHash: block52.HeaderHash}
	assert.Equal(t, rp.recover(bcStore), nil)

	hash, err := bcStore.GetBlockHash(5)
	assert.Equal(t, err, nil)
	assert.Equal(t, hash, block52.HeaderHash)

	hash, err = bcStore.GetBlockHash(4)
	assert.Equal(t, err, nil)
	assert.Equal(t, hash, block42.HeaderHash)
}
