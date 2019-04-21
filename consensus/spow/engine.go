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
	numOfBits := difficultyToNumOfBits(block.Header.Difficulty, block.Header.Height)

	E := big.NewInt(0).Exp(big.NewInt(2), numOfBits, nil)
	S := big.NewInt(0).Sub(E, big.NewInt(1))

	threads := engine.threads
	segmentInterval := uint64(5000) * uint64(numOfBits.Int64())

	hashArr := make([]uint64, uint64((threads + 4))*segmentInterval)
	nonceArr := make([]uint64, uint64((threads + 4))*segmentInterval)
	LEN := uint64(uint64((threads + 4)) * segmentInterval)
	once := &sync.Once{}
	timestampBegin := time.Now().Unix()
	bitsToNonceMap := make(map[uint64]uint64)
	sizeOfBitsToNonceMap := len(hashArr) * 50 
	refreshSize := 100000

	var pend sync.WaitGroup

	engine.log.Debug(" %d Threads, Buffer[%d] ", threads, segmentInterval)
	pend.Add(threads)

	for i := 0; i < threads; i++ {

		go func(id int) {
			defer pend.Done()

			header := block.Header.Clone()
			header.SecondWitness = []byte{}
			header.Witness = []byte(strconv.FormatUint(uint64(1234), 10))
			hash := header.Hash()
			A := hash.Big()
			slice := big.NewInt(0).And(A.Rsh(A, 96), S).Uint64()
			threadsID := uint64(id)

			arrIndex := uint64(0)
			arrIndex = uint64(id) * segmentInterval
			r := rand.New(rand.NewSource(time.Now().UnixNano()))

			if threadsID < uint64(threads - 1) || uint64(threads) == 1 {
				thisThreads := threadsID
				if uint64(threads) == 1 {
					for {
						if atomic.LoadInt32(&isNonceFound) != 0 {
							engine.log.Debug("exit mining as nonce is found bybreak")
							return
						}

						for nonce := uint64(0); nonce < segmentInterval; nonce++ {
							arrIndex = uint64(threadsID) * segmentInterval + uint64(100) + nonce
							Nonce := uint64(r.Int63n(int64(math.MaxUint64 / 2))) + nonce + thisThreads * segmentInterval
							header.Witness = []byte(strconv.FormatUint(Nonce, 10))
							hash = header.Hash()
							A = hash.Big()
							slice = big.NewInt(0).And(A.Rsh(A, 96), S).Uint64()
							hashArr[arrIndex] = slice
							nonceArr[arrIndex] = Nonce
						}

						Slice := uint64(0)
						Nonce := uint64(0)

						if atomic.LoadInt32(&isNonceFound) != 0 {
							engine.log.Debug("exit mining as nonce is found bybreak")

							return
						}

						for index := uint64(0); index < uint64(threads) * segmentInterval; index++ {
							if index > uint64(len(hashArr)) - 2 {
								engine.log.Debug("WRONG,%d,%d", index, len(hashArr))
							}

							if index > LEN - 1 {
								index = uint64(0)
							}
							Slice = hashArr[index]
							Nonce = nonceArr[index]

							if compareNonce, ok := bitsToNonceMap[Slice]; ok {
								if compareNonce != Nonce {
									once.Do(func() {
										engine.log.Info("Solution found, nonceA: %d, nonceB: %d, Map Size %d ", Nonce, compareNonce, len(bitsToNonceMap))
										engine.log.Info("solution time: %d (s)", time.Now().Unix()-timestampBegin)
										handleResults(block, results, stop, &isNonceFound, Nonce, compareNonce, engine.log)
									})
									return

								}
							} else {
								if Nonce > 0 {
									// refresh bitsToNonceMap if it is too big
									if len(bitsToNonceMap) > sizeOfBitsToNonceMap {
										counter := 0
										for bitsKey, _ := range bitsToNonceMap {
											if counter > refreshSize {
												break
											} 
											delete(bitsToNonceMap, bitsKey)
											counter++
										}
									}
									bitsToNonceMap[Slice] = Nonce									
								}
							}
						}

						select {
						case <-stop:
							return

						default:
							if atomic.LoadInt32(&isNonceFound) != 0 {
								engine.log.Debug("exit mining as nonce is found bybreak")
								return
							}
						}
					}

				} else {
					for {
						if atomic.LoadInt32(&isNonceFound) != 0 {
							engine.log.Debug("exit mining as nonce is found bybreak")
							return
						}

						for nonce := uint64(0); nonce < segmentInterval; nonce++ {
							arrIndex = uint64(threadsID)*segmentInterval + uint64(100) + nonce
							Nonce := uint64(r.Int63n(int64(math.MaxUint64/2))) + nonce + thisThreads*segmentInterval
							header.Witness = []byte(strconv.FormatUint(Nonce, 10))
							hash = header.Hash()
							A = hash.Big()
							slice = big.NewInt(0).And(A.Rsh(A, 96), S).Uint64()
							hashArr[arrIndex] = slice
							nonceArr[arrIndex] = Nonce
						}
					}
				}
			}

			if threadsID == uint64(threads - 1) && int64(threads) > 1 {
				thisThreads := threadsID
				for {
					Slice := uint64(0)
					Nonce := uint64(0)

					if atomic.LoadInt32(&isNonceFound) != 0 {
						engine.log.Debug("exit mining as nonce is found bybreak")
						return
					}

					for index := uint64(0); index < uint64(thisThreads + 1) * segmentInterval; index++ {
						if index > uint64(len(hashArr)) - 2 {
							engine.log.Debug("WRONG,%d,%d", index, len(hashArr))
						}

						if index > LEN - 1 {
							index = uint64(0)
						}
						Slice = hashArr[index]
						Nonce = nonceArr[index]

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
								// refresh bitsToNonceMap if it is too big
								if len(bitsToNonceMap) > sizeOfBitsToNonceMap {
									counter := 0
									for bitsKey, _ := range bitsToNonceMap {
										if counter > refreshSize {
											break
										} 
										delete(bitsToNonceMap, bitsKey)
										counter++
									}
								}
								bitsToNonceMap[Slice] = Nonce
							}
						}
					}

					select {
					case <-stop:
						return

					default:
						if atomic.LoadInt32(&isNonceFound) != 0 {
							engine.log.Debug("exit mining as nonce is found bybreak")
							return
						}
					}
				}
			}
		}(i)
	}
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
		engine.log.Info("invalid parent hash: %v", header.PreviousBlockHash)
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

	numOfBits := difficultyToNumOfBits(header.Difficulty, header.Height)

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

func difficultyToNumOfBits(difficulty *big.Int, height uint64) *big.Int {

	bigDiv := big.NewInt(int64(200000))
	var numOfBits = new(big.Int).Set(difficulty)
	numOfBits.Div(difficulty, bigDiv)

	if height > uint64(common.ForkHeight) && numOfBits.Cmp(big.NewInt(int64(70))) > 0 {
		numOfBits = big.NewInt(int64(70))
	} 

	if height <= uint64(common.ForkHeight) && numOfBits.Cmp(big.NewInt(int64(50))) > 0 {
		numOfBits = big.NewInt(int64(50))
	} 

	if numOfBits.Cmp(big.NewInt(int64(1))) < 0 {
		numOfBits = big.NewInt(int64(1))
	}
	return numOfBits
}
