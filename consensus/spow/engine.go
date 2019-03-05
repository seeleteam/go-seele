/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package spow

import (
	"math"
	"math/rand"
	"math/big"
	"strconv"
	"time"
	"runtime"
	"sync"
	"os"
	"bytes"
	"encoding/binary"
	"path/filepath"
	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/consensus/utils"
	"github.com/seeleteam/go-seele/rpc"
)

var (
	// the number of hashes for hash collison 
	hashPoolSize = uint64(33000000)
	pack = uint64(600000)
)


type HashItem struct {
	Hash  common.Hash 
	Nonce uint64
}

// Engine provides the consensus operations based on POW.
type SpowEngine struct {
	threads        int
	log            *log.SeeleLog
	hashrate       metrics.Meter
	hashPoolDB     database.Database
	hashPoolDBPath string
}

func NewSpowEngine(threads int) *SpowEngine {

	baseDir := common.GetTempFolder()
	datasetDir := filepath.Join(baseDir, "datasets")
	return &SpowEngine{
		threads:        threads,
		log:            log.GetLogger("spow_engine"),
		hashrate:       metrics.NewMeter(),
		hashPoolDBPath: datasetDir,
	}
}

func (engine *SpowEngine) SetThreads(threads int) {
	if threads <= 0 {
		engine.threads = runtime.NumCPU()
	} else {
		engine.threads = threads
	}
}

func (engine *SpowEngine) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{
		{
			Namespace: "spow",
			Version:   "1.0",
			Service:   &API{engine},
			Public:    true,
		},
	}
}

func (engine *SpowEngine) Prepare(reader consensus.ChainReader, header *types.BlockHeader) error {
	parent := reader.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return consensus.ErrBlockInvalidParentHash
	}

	header.Difficulty = utils.GetDifficult(header.CreateTimestamp.Uint64(), parent)

	return nil
}

func (engine *SpowEngine) Seal(reader consensus.ChainReader, block *types.Block, stop <-chan struct{}, results chan<- *types.Block) error {

	var err error
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// make sure beginNonce is not too big
	beginNonce := uint64(r.Int63n(int64(math.MaxUint64 / 2)))

	if beginNonce + hashPoolSize < math.MaxUint64 {

		// new hashPool database
		if engine.hashPoolDB, err = leveldb.NewLevelDB(engine.hashPoolDBPath); err != nil {
			engine.log.Error("spow err: failed to create hashPool DB, %s", err)
			return err
		}

		hashPackIndex, packsPerThread := engine.generateHashPackIndex()

		// generate hashPool
		if err = engine.generateHashPool(block, beginNonce, hashPackIndex, packsPerThread); err != nil {
			engine.log.Error("spow err: failed to generate hashPool, %s", err)
			return err
		}

		go engine.startCollision(block, results, stop, beginNonce, hashPackIndex)
	}

	return nil
		
}

func (engine *SpowEngine) generateHashPackIndex() ([]uint64, uint64) {
	// generate the index list for hashPacks in the database
	var hashPackIndex []uint64

	threads := engine.threads
	hashesPerThread := hashPoolSize
	if threads != 0 {
		hashesPerThread = hashPoolSize / uint64(threads)
	}
	packsPerThread := hashesPerThread / pack
	begin := uint64(0)
	for i := 0; i < threads; i++ {
		for j := uint64(0); j < packsPerThread; j++ {
			hashPackIndex = append(hashPackIndex, begin)
			begin += pack
		} 
		if hashesPerThread > pack * packsPerThread {
			hashPackIndex = append(hashPackIndex, begin)
			begin += hashesPerThread - pack * packsPerThread
		}
	}

	// mark the end
	hashPackIndex = append(hashPackIndex, begin)
	if hashesPerThread > pack * packsPerThread {
		packsPerThread += 1
	}

	return hashPackIndex, packsPerThread
}


func (engine *SpowEngine) generateHashPool(block *types.Block, beginNonce uint64, hashPackIndex []uint64, packsPerThread uint64) error {

	threads := engine.threads
	
	// generate the hashPool concurrently
	var err error
	var pend sync.WaitGroup
	pend.Add(threads)

	for i := 0; i < threads; i++ {
		go func(id int) {
			defer pend.Done()

			header := block.Header.Clone()
				
			// Calculate the dataset segment
			counter := uint64(0)
			for counter < packsPerThread {

				// create hash pack  
				var hashPack []*HashItem 
				idx := uint64(id) * packsPerThread + counter
				for nonce := beginNonce + hashPackIndex[idx]; nonce < beginNonce + hashPackIndex[idx + 1]; nonce++ {
					header.Witness = []byte(strconv.FormatUint(nonce, 10))
					header.SecondWitness = []byte{}
					hash := header.Hash()					
					hashItem := &HashItem{
						Hash: hash,
						Nonce: nonce,
					}
					hashPack = append(hashPack, hashItem)
				}	

				// batch commit the hashes to the database
				if len(hashPack) > 0 {
					batch := engine.hashPoolDB.NewBatch()
					encoded := make([]byte, 8)
					binary.BigEndian.PutUint64(encoded, idx)
					batch.Put(encoded, common.SerializePanic(hashPack))
					err = batch.Commit()
					if err != nil {
						engine.log.Warn("failed to store hashPack in database, err %s", err)
					}
				}

				counter++					  
			}
		}(i)
	}
	// Wait for all the generators to finish and return
	pend.Wait()
	return err

}

func (engine *SpowEngine) startCollision(block *types.Block, results chan<- *types.Block, stop <-chan struct{}, beginNonce uint64, hashPackIndex []uint64) {
	
	defer engine.removeDB()

miner:
	for i := 0; i < len(hashPackIndex) - 1; i++ {
		baseHashPack, err := engine.getHashPack(i)
		if err != nil {
			break miner
		}

		// baseHashPack compare with itself
		for k := uint64(0); k < hashPackIndex[i + 1] - hashPackIndex[i]; k++ {
			for n := k + 1; n < hashPackIndex[i + 1] - hashPackIndex[i]; n++ {
				isFound := isPair(baseHashPack[k].Hash, baseHashPack[n].Hash, block.Header.Difficulty)
				// nonce pair is found
				if isFound {
					engine.log.Info("nonceA: %d, hashA: %s, nonceB: %d, hashB: %s", baseHashPack[k].Nonce, baseHashPack[k].Hash.Hex(), baseHashPack[n].Nonce, baseHashPack[n].Hash.Hex())
					engine.removeDB()
					handleResults(block, results, stop, baseHashPack[k].Nonce, baseHashPack[n].Nonce, engine.log)

					break miner
				}
			}
		} 	

		// compare base hash pack with other hash packs
		for j := i + 1; j < len(hashPackIndex) - 1; j++ {
			compareHashPack, err := engine.getHashPack(j)
			if err != nil {
				break miner
			}

			for k := uint64(0); k < hashPackIndex[i + 1] - hashPackIndex[i]; k++ {
				for n := uint64(0); n < hashPackIndex[j + 1] - hashPackIndex[j]; n++ {
					isFound := isPair(baseHashPack[k].Hash, compareHashPack[n].Hash, block.Header.Difficulty)
					// nonce pair is found
					if isFound {
						engine.log.Info("nonceA: %d, hashA: %s, nonceB: %d, hashB: %s", baseHashPack[k].Nonce, baseHashPack[k].Hash.Hex(), compareHashPack[n].Nonce, compareHashPack[n].Hash.Hex())
						engine.removeDB()
						handleResults(block, results, stop, baseHashPack[k].Nonce, compareHashPack[n].Nonce, engine.log)
	
						break miner
					}
				}
			} 	
		} 
	}

}

func (engine *SpowEngine) removeDB() {

	if engine.hashPoolDB != nil {
		engine.hashPoolDB.Close()
		os.RemoveAll(engine.hashPoolDBPath)
		engine.hashPoolDB = nil
	}

	return

}


func (engine *SpowEngine) getHashPack(id int) ([]*HashItem, error) {

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, uint64(id))
	value, err := engine.hashPoolDB.Get(key)
	if err != nil {
		return nil, err
	}

	var hashPack []*HashItem
	if err = common.Deserialize(value, &hashPack); err != nil {
		return nil, err
	}

	return hashPack, nil
}


func handleResults(block *types.Block, result chan<- *types.Block, abort <-chan struct{}, nonceA uint64, nonceB uint64, log *log.SeeleLog) {

	// put the nonce pair in the block
	header := block.Header.Clone()
	header.Witness = []byte(strconv.FormatUint(nonceA, 10))
	header.SecondWitness = []byte(strconv.FormatUint(nonceB, 10))
	block.Header = header
	block.HeaderHash = header.Hash()

	select {
	case <-abort:
		logAbort(log)
	case result <- block:
		log.Info("nonce finding succeeded")
	}

	return
}

// logAbort logs the info that nonce finding is aborted
func logAbort(log *log.SeeleLog) {
	log.Info("nonce finding aborted")
}


// ValidateHeader validates the specified header and returns error if validation failed.
func (engine *SpowEngine) VerifyHeader(reader consensus.ChainReader, header *types.BlockHeader) error {
	parent := reader.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return consensus.ErrBlockInvalidParentHash
	}

	if err := utils.VerifyHeaderCommon(header, parent); err != nil {
		return err
	}

	if err := verifyPair(header); err != nil {
		return err
	}

	return nil
}


func verifyPair(header *types.BlockHeader) error {

	NewHeader := header.Clone()
	// two nonces must be different
	if (bytes.Equal(NewHeader.Witness, NewHeader.SecondWitness)) {
		return consensus.ErrBlockNonceInvalid
	}
	nonceB := NewHeader.SecondWitness
	NewHeader.SecondWitness = []byte{}
	hashA := NewHeader.Hash()
	NewHeader.Witness = nonceB
	hashB := NewHeader.Hash()

	numOfBits := header.Difficulty
	
	if p := isPair(hashA, hashB, numOfBits); p == false {
		return consensus.ErrBlockNonceInvalid
	}

	return nil
}

func isPair(hashA common.Hash, hashB common.Hash, numOfBits *big.Int) bool {

	A := hashA.Big()
	B := hashB.Big()
	E := big.NewInt(0).Exp(big.NewInt(2), numOfBits, nil)
	S := big.NewInt(0).Sub(E, big.NewInt(1))
	if big.NewInt(0).And(A.Rsh(A, 96), S).Cmp(big.NewInt(0).And(B.Rsh(B, 96), S)) == 0 {
		return true
	} else {
		return false
	}
}