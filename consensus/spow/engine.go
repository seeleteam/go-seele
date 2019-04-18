/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package spow

import (
	"bytes"
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/utils"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/rpc"
)

var (
	// the number of hashes for hash collison
	baseHashPoolSize = uint64(100000)
)

type HashItem struct {
	//Hash  common.Hash
	Slice uint64
	Nonce uint64
}

// Engine provides the consensus operations based on SPOW.
type SpowEngine struct {
	threads        int
	log            *log.SeeleLog
	hashrate       metrics.Meter
	hashPoolDB     database.Database
	hashPoolDBPath string
	lock           sync.Mutex
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
	if block.Header.Difficulty.Uint64() > 5200000 {
		hashPoolSize = baseHashPoolSize * uint64(1<<((block.Header.Difficulty.Uint64()-5200000)/400000))
	} else {
		hashPoolSize = baseHashPoolSize >> uint64((5200000-block.Header.Difficulty.Uint64())/400000)
	}

	if beginNonce+hashPoolSize < math.MaxUint64 {

		threads := engine.threads
		hashesPerThread := hashPoolSize
		if threads != 0 {
			hashesPerThread = hashPoolSize / uint64(threads)
		}

		select {
		case <-stop:
			return nil

		default:
			go engine.startCollision(block, results, stop, beginNonce, hashesPerThread)
		}
	}

	return nil

}

/*use arrays and random read value*/
func (engine *SpowEngine) startCollision(block *types.Block, results chan<- *types.Block, stop <-chan struct{}, beginNonce uint64, hashesPerThread uint64) {

	var isNonceFound int32
	numOfBits := difficultyToNumOfBits(block.Header.Difficulty)
	bitsToNonceMap := make(map[uint64]uint64)
	E := big.NewInt(0).Exp(big.NewInt(2), numOfBits, nil)
	S := big.NewInt(0).Sub(E, big.NewInt(1))

	threads := engine.threads

	const segmentInterval = 500000
	hashArr := make([]uint64, (threads+1)*segmentInterval)
	nonceArr := make([]uint64, (threads+1)*segmentInterval)
	once := &sync.Once{}
	timestampBegin := time.Now().Unix()

	var pend sync.WaitGroup

	pend.Add(threads)

	for i := 0; i < threads; i++ {

		go func(id int) {
			defer pend.Done()
			if id < threads-1 {

				header := block.Header.Clone()
				header.SecondWitness = []byte{}
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				starter := uint64(r.Int63n(int64(math.MaxUint64/2))) * uint64(r.Int63n(int64(math.MaxUint64/2))) * uint64(id+1)
				uppBound := starter + uint64(segmentInterval)

				header.Witness = []byte(strconv.FormatUint(uint64(1234), 10))
				hash := header.Hash()
				A := hash.Big()
				slice := big.NewInt(0).And(A.Rsh(A, 96), S).Uint64()
				arrIndex := uint64(0)
				for {

					r = rand.New(rand.NewSource(time.Now().UnixNano()))
					starter = uint64(r.Int63n(int64(math.MaxUint64/2)))*uint64(r.Int63n(int64(math.MaxUint64/2)))*uint64(id+1) + starter

					uppBound = starter + uint64(segmentInterval)

					k := 0
					for nonce := starter; nonce < uppBound; nonce++ {

						select {
						case <-stop:
							logAbort(engine.log)
							return
						default:
							if atomic.LoadInt32(&isNonceFound) != 0 {
								engine.log.Debug("exit mining as nonce is found bybreak")

								return
							}

							header.Witness = []byte(strconv.FormatUint(nonce, 10))

							hash = header.Hash()

							A = hash.Big()
							slice = big.NewInt(0).And(A.Rsh(A, 96), S).Uint64()

							arrIndex = uint64(id*segmentInterval + k) // in case the array index exceeds

							hashArr[arrIndex] = slice
							nonceArr[arrIndex] = nonce

							k = k + 1

						}

					}
					time.Sleep(time.Millisecond * 300)
				}
			} else {
				index := int(0)
				Slice := uint64(0)
				Nonce := uint64(0)
				for {
					if atomic.LoadInt32(&isNonceFound) != 0 {

						engine.log.Debug("exit mining as nonce is found by break")

						return
					}

					Slice = hashArr[index]
					Nonce = nonceArr[index]

					if index >= threads*segmentInterval { //rescan the array
						index = 0
					} else {
						index = index + 1
					}

					if compareNonce, ok := bitsToNonceMap[Slice]; ok {

						if compareNonce != Nonce {
							once.Do(func() {
								engine.log.Info("Find solution,nonceA: %d, nonceB: %d, Map Size %d ", Nonce, compareNonce, len(bitsToNonceMap))
								engine.log.Info("solution time: %d (s)", time.Now().Unix()-timestampBegin)
								handleResults(block, results, stop, &isNonceFound, Nonce, compareNonce, engine.log)

							})
							return
						}
					} else {
						if Nonce > 0 {
							bitsToNonceMap[Slice] = Nonce
						}

					}
				}

			}

		}(i)
	}
	// Wait for all the threads to finish and return

	pend.Wait()
}
func handleResults(block *types.Block, result chan<- *types.Block, abort <-chan struct{}, isNonceFound *int32, nonceA uint64, nonceB uint64, log *log.SeeleLog) {

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
		atomic.StoreInt32(isNonceFound, 1)
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
	if bytes.Equal(NewHeader.Witness, NewHeader.SecondWitness) {
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
	if numOfBits.Cmp(big.NewInt(int64(70))) > 0 {
		numOfBits = big.NewInt(int64(70))
	}

	if numOfBits.Cmp(big.NewInt(int64(1))) < 0 {
		numOfBits = big.NewInt(int64(1))
	}
	return numOfBits
}
