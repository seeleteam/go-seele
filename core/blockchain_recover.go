/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
)

var rpLog = log.GetLogger("recoveryPoint")

// recoveryPoint is used for blockchain recovery in case of program crashed when write a block.
type recoveryPoint struct {
	WritingBlockHash           common.Hash // block hash that was writing to blockchain.
	WritingBlockHeight         uint64      // block height that was writing to blockchain.
	PreviousCanonicalBlockHash common.Hash // overwritten block hash once the writing block is new HEAD in canonical chain.
	PreviousHeadBlockHash      common.Hash // current HEAD block hash when write a block.
	LargerHeight               uint64      // Record the larger height block that to be removed from canonical chain.
	StaleHash                  common.Hash // Record the stale block hash for overwrite in canonical chain.

	file string
}

func loadRecoveryPoint(file string) (*recoveryPoint, error) {
	rp := recoveryPoint{
		file: file,
	}

	if len(file) == 0 || !common.FileOrFolderExists(file) {
		return &rp, nil
	}

	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		rpLog.Error("Failed to read bytes from recovery point file, %v", err.Error())
		return &rp, err
	}

	if err = json.Unmarshal(bytes, &rp); err != nil {
		rpLog.Warn("Failed to unmarshal encoded JSON data to recovery point info, file = %v, error = %v", file, err.Error())
		rp.serialize()
	}

	return &rp, nil
}

func (rp *recoveryPoint) recover(bcStore store.BlockchainStore) error {
	saved := true

	// recover the previous HEAD block hash.
	if !rp.PreviousHeadBlockHash.IsEmpty() {
		if err := bcStore.PutHeadBlockHash(rp.PreviousHeadBlockHash); err != nil {
			rpLog.Error("Failed to recover HEAD block hash, hash = %v, error = %v", rp.PreviousCanonicalBlockHash.ToHex(), err.Error())
			return err
		}

		rp.PreviousHeadBlockHash = common.EmptyHash
		rpLog.Info("HEAD block hash recovered successfully")
	}

	// recover the previous block hash in canonical chain.
	if rp.WritingBlockHeight > 0 && !rp.PreviousCanonicalBlockHash.IsEmpty() {
		if err := bcStore.PutBlockHash(rp.WritingBlockHeight, rp.PreviousCanonicalBlockHash); err != nil {
			rpLog.Error("Failed to recover the block hash by height in canonical chain, height = %v, hash = %v, error = %v", rp.LargerHeight, rp.PreviousCanonicalBlockHash, err.Error())
			return err
		}

		rp.PreviousCanonicalBlockHash = common.EmptyHash
		rpLog.Info("the block hash by height in canonical chain recovered successfully")
	}

	// delete the crashed block.
	if !rp.WritingBlockHash.IsEmpty() {
		if err := bcStore.DeleteBlock(rp.WritingBlockHash); err != nil {
			rpLog.Error("Failed to delete the crashed block, hash = %v, error = %v", rp.WritingBlockHash, err.Error())
			return err
		}

		rp.WritingBlockHash = common.EmptyHash
		saved = false
		rpLog.Info("the crashed block deleted successfully")
	}

	// go on to delete larger height blocks from canonical chain.
	if saved && rp.LargerHeight > 0 {
		if err := deleteLargerHeightBlocks(bcStore, rp.LargerHeight, nil); err != nil {
			rpLog.Error("Failed to delete the larger height blocks in canonical chain, height = %v, error = %v", rp.LargerHeight, err.Error())
			return err
		}

		rpLog.Info("the larger height blocks in canonical chain deleted successfully")
	}

	rp.LargerHeight = 0

	// go on to overwrite stale blocks in canonical chain.
	if saved && !rp.StaleHash.IsEmpty() {
		if err := overwriteStaleBlocks(bcStore, rp.StaleHash, nil); err != nil {
			rpLog.Error("Failed to overwrite the stale blocks in canonical chain, hash = %v, error = %v", rp.StaleHash, err.Error())
			return err
		}

		rpLog.Info("stale blocks in canonical chain overwrited successfully")
	}

	rp.StaleHash = common.EmptyHash

	rp.serialize()

	return nil
}

func (rp *recoveryPoint) serialize() {
	// do nothing if file is empty.
	// Generally, UT could use empty file name to ignore the recovery point mechanism.
	if len(rp.file) == 0 {
		return
	}

	encoded, err := json.MarshalIndent(rp, "", "\t")
	if err != nil {
		// just log the error so as not to block the blockchain initialization.
		rpLog.Warn("Failed to marshal recovery point info to JSON data, rp = %+v, error = %v", *rp, err.Error())
		return
	}

	if err := ioutil.WriteFile(rp.file, encoded, os.ModePerm); err != nil {
		// just log the error so as not to block the blockchain initialization.
		rpLog.Warn("Failed to write recovery point JSON data to file, file = %v, error = %v", rp.file, err.Error())
	}
}

func (rp *recoveryPoint) onPutBlockStart(block *types.Block, bcStore store.BlockchainStore, isHead bool) error {
	rp.WritingBlockHash = block.HeaderHash
	rp.WritingBlockHeight = block.Header.Height

	// the block of specified height may not exist in canonical chain.
	if hash, err := bcStore.GetBlockHash(rp.WritingBlockHeight); err == nil {
		rp.PreviousCanonicalBlockHash = hash
	} else {
		rp.PreviousCanonicalBlockHash = common.EmptyHash
	}

	// HEAD block hash must exist
	hash, err := bcStore.GetHeadBlockHash()
	if err != nil {
		rpLog.Error("Failed to get HEAD block hash onPutBlockStart, %v", err.Error())
		return err
	}

	rp.PreviousHeadBlockHash = hash

	if isHead {
		rp.LargerHeight = block.Header.Height + 1
		rp.StaleHash = block.Header.PreviousBlockHash
	} else {
		rp.LargerHeight = 0
		rp.StaleHash = common.EmptyHash
	}

	rp.serialize()

	return nil
}

func (rp *recoveryPoint) onPutBlockEnd() {
	rp.PreviousHeadBlockHash = common.EmptyHash
	rp.WritingBlockHeight = 0
	rp.PreviousCanonicalBlockHash = common.EmptyHash
	rp.WritingBlockHash = common.EmptyHash

	rp.serialize()
}

func (rp *recoveryPoint) onDeleteLargerHeightBlocks(height uint64) {
	rp.LargerHeight = height
	rp.serialize()
}

func (rp *recoveryPoint) onOverwriteStaleBlocks(hash common.Hash) {
	rp.StaleHash = hash
	rp.serialize()
}
