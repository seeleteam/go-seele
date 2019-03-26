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
	"bytes"

	"github.com/seeleteam/go-seele/common"
	"github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/consensus/utils"
	"github.com/seeleteam/go-seele/rpc"
)

var (
	// the number of hashes for hash collison 
	baseHashPoolSize = uint64(100000)
)


type HashItem struct {
	Hash  common.Hash 
	Nonce uint64
}

// Engine provides the consensus operations based on SPOW.
type SpowEngine struct {
	threads        int
	log            *log.SeeleLog
	hashrate       metrics.Meter
	hashPoolDB     database.Database
	hashPoolDBPath string
}

func NewSpowEngine(threads int, folder string) *SpowEngine {

	return &SpowEngine{
		threads:        threads,
		log:            log.GetLogger("spow_engine"),
		hashrate:       metrics.NewMeter(),
		hashPoolDBPath: folder,
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
			Namespace: "miner",
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

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// make sure beginNonce is not too big
	beginNonce := uint64(r.Int63n(int64(math.MaxUint64 / 2)))

	var hashPoolSize uint64
	if (block.Header.Difficulty.Uint64() > 5200000) {
		hashPoolSize = baseHashPoolSize * uint64(1 << (( block.Header.Difficulty.Uint64() - 5200000) / 400000))
	} else {
		hashPoolSize = baseHashPoolSize >> uint64((5200000 - block.Header.Difficulty.Uint64()) / 400000)
	}
	
	if hashPoolSize > uint64(400000000) {
		hashPoolSize = uint64(400000000)
	}
	if beginNonce + hashPoolSize < math.MaxUint64 {

		threads := engine.threads
		hashesPerThread := hashPoolSize
		if threads != 0 {
			hashesPerThread = hashPoolSize / uint64(threads)
		}

		// generate hashPool
		engine.log.Info("start generating hashpool, %d", hashPoolSize)
		hashPack, err := engine.generateHashPool(block, stop, beginNonce, hashesPerThread)
		engine.log.Info("Hashpool is generated!")
		if err != nil {
			engine.log.Error("spow err: failed to generate hashPool, %s", err)
			return err
		}

		select {
		case <-stop:
			return nil

		default:
			go engine.startCollision(block, results, stop, beginNonce, hashPack)
		}
	}

	return nil
		
}

func (engine *SpowEngine) generateHashPool(block *types.Block, stop <-chan struct{}, beginNonce uint64, hashesPerThread uint64) ([][]*HashItem, error) {

	threads := engine.threads
	
	// generate the hashPool concurrently
	var err error
	var pend sync.WaitGroup
	hashPack := make([][]*HashItem, threads) 

	pend.Add(threads)

	for i := 0; i < threads; i++ {
		go func(id int) {
			defer pend.Done()

			header := block.Header.Clone()
				
			// Calculate the segment in each thread
			for nonce := beginNonce + uint64(id) * hashesPerThread; nonce < beginNonce + uint64(id + 1) * hashesPerThread; nonce++ {
				select {
				case <-stop:
					logAbort(engine.log)
					return
		
				default:
					header.Witness = []byte(strconv.FormatUint(nonce, 10))
					header.SecondWitness = []byte{}
					hash := header.Hash()					
					hashItem := &HashItem{
						Hash: hash,
						Nonce: nonce,
					}
					hashPack[id] = append(hashPack[id], hashItem)
				}
			}	
		}(i)
	}
	// Wait for all the generators to finish and return
	pend.Wait()
	return hashPack, err

}


func (engine *SpowEngine) startCollision(block *types.Block, results chan<- *types.Block, stop <-chan struct{}, beginNonce uint64, hashPack [][]*HashItem) {

	numOfBits := difficultyToNumOfBits(block.Header.Difficulty)

miner:
	for i := 0; i < len(hashPack); i++ {
		baseHashPack := hashPack[i]

		// baseHashPack compare with itself
		for k := 0; k < len(baseHashPack); k++ {
			for n := k + 1; n < len(baseHashPack); n++ {
				select {
				case <-stop:
					logAbort(engine.log)
					break miner
	
				default:
					isFound := isPair(baseHashPack[k].Hash, baseHashPack[n].Hash, numOfBits)
					// nonce pair is found
					if isFound {
						engine.log.Info("nonceA: %d, hashA: %s, nonceB: %d, hashB: %s", baseHashPack[k].Nonce, baseHashPack[k].Hash.Hex(), baseHashPack[n].Nonce, baseHashPack[n].Hash.Hex())
						handleResults(block, results, stop, baseHashPack[k].Nonce, baseHashPack[n].Nonce, engine.log)

						break miner
					}
				}
			}
		} 	

		// compare base hash pack with other hash packs
		for j := i + 1; j < len(hashPack); j++ {
			compareHashPack := hashPack[j]

			for k := 0; k < len(baseHashPack); k++ {
				for n := 0; n < len(compareHashPack); n++ {
					select {
					case <-stop:
						logAbort(engine.log)
						break miner
		
					default:
						isFound := isPair(baseHashPack[k].Hash, compareHashPack[n].Hash, numOfBits)
						// nonce pair is found
						if isFound {
							engine.log.Info("nonceA: %d, hashA: %s, nonceB: %d, hashB: %s", baseHashPack[k].Nonce, baseHashPack[k].Hash.Hex(), compareHashPack[n].Nonce, compareHashPack[n].Hash.Hex())
							handleResults(block, results, stop, baseHashPack[k].Nonce, compareHashPack[n].Nonce, engine.log)
	
							break miner
						}
					}
				}
			} 	
		} 
	}

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
	
	numOfBits := difficultyToNumOfBits(header.Difficulty)

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

func difficultyToNumOfBits(difficulty *big.Int) *big.Int {

	bigDiv := big.NewInt(int64(200000))
	var numOfBits = new(big.Int).Set(difficulty)
	numOfBits.Div(difficulty, bigDiv)
	if numOfBits.Cmp(big.NewInt(int64(50))) > 0 {
		numOfBits = big.NewInt(int64(50))
	}

	if numOfBits.Cmp(big.NewInt(int64(1))) < 0 {
		numOfBits = big.NewInt(int64(1))
	}
	return numOfBits
}