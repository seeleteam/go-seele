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
	"strings"
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
	"gonum.org/v1/gonum/mat"
)

var (
	// the number of hashes for hash collison
	baseHashPoolSize = uint64(100000)
	// Hadamard's bound for the absolute determinant of an 𝑛×𝑛 0-1 matrix is {(n + 1)^[(n+1)/2]} / 2^n
	maxDet30x30 = new(big.Int).Mul(big.NewInt(2), new(big.Int).Exp(big.NewInt(10), big.NewInt(13), big.NewInt(0)))
	matrixDim   = int(30)
	RestTime    = 50 * time.Millisecond
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

	// fork control
	if block.Header.Height >= common.SecondForkHeight || (block.Header.Creator.Shard() == uint(1) && block.Header.Height > common.ForkHeight) {
		return engine.MSeal(reader, block, stop, results)
	}

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

	hashArr := make([]uint64, uint64((threads+4))*segmentInterval)
	nonceArr := make([]uint64, uint64((threads+4))*segmentInterval)
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

			if threadsID < uint64(threads-1) || uint64(threads) == 1 {
				thisThreads := threadsID
				if uint64(threads) == 1 {
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

						Slice := uint64(0)
						Nonce := uint64(0)

						if atomic.LoadInt32(&isNonceFound) != 0 {
							engine.log.Debug("exit mining as nonce is found bybreak")

							return
						}

						for index := uint64(0); index < uint64(threads)*segmentInterval; index++ {
							if index > uint64(len(hashArr))-2 {
								engine.log.Debug("WRONG,%d,%d", index, len(hashArr))
							}

							if index > LEN-1 {
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

			if threadsID == uint64(threads-1) && int64(threads) > 1 {
				thisThreads := threadsID
				for {
					Slice := uint64(0)
					Nonce := uint64(0)

					if atomic.LoadInt32(&isNonceFound) != 0 {
						engine.log.Debug("exit mining as nonce is found bybreak")
						return
					}

					for index := uint64(0); index < uint64(thisThreads+1)*segmentInterval; index++ {
						if index > uint64(len(hashArr))-2 {
							engine.log.Debug("WRONG,%d,%d", index, len(hashArr))
						}

						if index > LEN-1 {
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
	time.Sleep(RestTime)
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

	if header.Height >= common.SecondForkHeight || (header.Creator.Shard() == uint(1) && header.Height > common.ForkHeight) {
		if err := engine.verifyTarget(header); err != nil {
			return err
		}
	} else {
		if err := verifyPair(header); err != nil {
			return err
		}
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

func (engine *SpowEngine) MSeal(reader consensus.ChainReader, block *types.Block, stop <-chan struct{}, results chan<- *types.Block) error {
	threads := engine.threads

	var step uint64
	var seed uint64
	if threads != 0 {
		step = math.MaxUint64 / uint64(threads)
	}

	var isNonceFound int32
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	once := &sync.Once{}
	for i := 0; i < threads; i++ {
		if threads == 1 {
			seed = r.Uint64()
		} else {
			seed = uint64(r.Int63n(int64(step)))
		}
		tSeed := seed + uint64(i)*step
		var min uint64
		var max uint64
		min = uint64(i) * step

		if i != threads-1 {
			max = min + step - 1
		} else {
			max = math.MaxUint64
		}

		go func(tseed uint64, tmin uint64, tmax uint64) {
			engine.MStartMining(block, tseed, tmin, tmax, results, stop, &isNonceFound, once, engine.hashrate, engine.log)
		}(tSeed, min, max)
	}

	return nil
}

func (engine *SpowEngine) MStartMining(block *types.Block, seed uint64, min uint64, max uint64, result chan<- *types.Block, abort <-chan struct{},
	isNonceFound *int32, once *sync.Once, hashrate metrics.Meter, log *log.SeeleLog) {
	var nonce = seed
	var hashInt big.Int
	var caltimes = int64(0)
	var isBegining = true
	target := getMiningTarget(block.Header.Difficulty)
	nonZeroCountTarget := getNonZeroCountTarget(matrixDim)
	header := block.Header.Clone()
	dim := matrixDim
	matrix := mat.NewDense(dim, 256, nil)

miner:
	for {
		select {
		case <-abort:
			logAbort(log)
			hashrate.Mark(caltimes)
			break miner

		default:
			if atomic.LoadInt32(isNonceFound) != 0 {
				log.Debug("exit mining as nonce is found by other threads")
				break miner
			}

			caltimes++
			if caltimes == 0x7FFF {
				hashrate.Mark(caltimes)
				caltimes = 0
			}

			header.Witness = []byte(strconv.FormatUint(nonce, 10))
			hash := header.Hash()
			hashInt.SetBytes(hash.Bytes())

			// isBegining: fresh start or nonce reverse: make sure nonce verify is right
			if isBegining == true { // there is no matrix yet
				if nonce+uint64(dim) >= max { // reach out of tail, reverse
					nonce = min
				}
				header.Witness = []byte(strconv.FormatUint(nonce, 10))
				hash = header.Hash()
				matrix = newMatrix(header, nonce, dim, log)
				isBegining = false
			} else {
				if nonce+uint64(dim) >= max { // reach out of tail, reverse
					nonce = min
				}
				matrix = submatCopyByRow(header, matrix, 1, dim, nonce)
				header.Witness = []byte(strconv.FormatUint(nonce-uint64(dim-1), 10))
				hash = header.Hash()
			}
			res, count := calDetmLoopForMining(matrix, dim, target, log)
			restInt := int64(res)
			restBig := big.NewInt(restInt)

			// found
			if restBig.Cmp(target) >= 0 && count >= nonZeroCountTarget {
				once.Do(func() {
					block.Header = header
					block.HeaderHash = hash

					select {
					case <-abort:
						logAbort(log)
					case result <- block:
						atomic.StoreInt32(isNonceFound, 1)
						log.Debug("found det:%d", restBig)
						log.Debug("target:%d", target)
						log.Debug("times2try:%d", caltimes)
					}
				})
				break miner
			}
			// outage
			if nonce == seed-1 {
				select {
				case <-abort:
					logAbort(log)
				case result <- nil:
					log.Warn("nonce finding outage")
				}

				break miner
			}
			nonce++
		}
	}
}

func (engine *SpowEngine) verifyTarget(header *types.BlockHeader) error {
	dim := matrixDim
	NewHeader := header.Clone()
	nonceUint64, err := strconv.ParseUint(string(NewHeader.Witness), 10, 64)
	if err != nil {
		return err
	}
	matrix := newMatrix(header, nonceUint64, dim, engine.log)
	res, count := calDetmLoopForVerification(matrix, dim, engine.log)
	restInt := int64(res)
	restBig := big.NewInt(restInt)
	target := getMiningTarget(header.Difficulty)
	if restBig.Cmp(target) < 0 || count < getNonZeroCountTarget(dim) {
		return consensus.ErrBlockNonceInvalid
	}
	return nil
}

// getMiningTarget returns the mining target for the specified difficulty.
func getMiningTarget(difficulty *big.Int) *big.Int {
	// 65: when switch from spow to mpow, diff is 11M, for mpow the test data show 80M is stable for block time (10ish second)
	target := new(big.Int).Mul(difficulty, big.NewInt(65))
	if target.Cmp(maxDet30x30) > 0 {
		return maxDet30x30
	}
	return target
}

func getNonZeroCountTarget(matrixDim int) int {
	return (256-matrixDim)/2 + matrixDim/5
}

func newMatrix(headerTarget *types.BlockHeader, seedNonce uint64, dim int, log *log.SeeleLog) *mat.Dense {
	header := headerTarget.Clone()
	nonce := seedNonce
	matrix := mat.NewDense(dim, 256, nil)
	for i := 0; i < dim; i++ {
		header.Witness = []byte(strconv.FormatUint(nonce, 10))
		header.SecondWitness = []byte{}
		hash := header.Hash()
		col, isLeg := getBinaryArray(hash.String())
		if isLeg == false {
			return nil
		}
		matrix.SetRow(i, col)
		nonce++
	}
	return matrix
}

func calDetm(matrix *mat.Dense, dim int, log *log.SeeleLog) float64 {
	submatrix := mat.NewDense(dim, dim, nil)
	submatrix.Copy(matrix)
	log.Debug("\n%0.1v\n\n", mat.Formatted(submatrix))
	det := mat.Det(submatrix)
	log.Debug("try det:%f\n", det)
	return det
}

// matrix is dim x 256
// calDetmLoop will loop from 0 to 256 - dim
func calDetmLoopForMining(matrix *mat.Dense, dim int, target *big.Int, log *log.SeeleLog) (float64, int) {
	var ret = float64(0)
	var nonZerosCount = int(0)
	nonZeroCountTarget := getNonZeroCountTarget(dim)
	submatrix := mat.NewDense(dim, dim, nil)
	// Check if the input matrix has a chance to have a submatrix whose determinant is greater than the target.
	// (Let's call such submatrix as "great submatrix").
	// In addition, such great great submatrix must be the 119th non-zero determinant in the input matrix.
	// The exact position of the 119th non-zero determinant is unknown at the moment,
	// and its computation is heavy because we have to calculate at least 119 determinants.
	// Thus, we compute the determinants of last N submatrices.
	// If there is no great submatrix in them, we just stop.
	// There is some optimal number to search, which balances the possibility of great submatrix and hashing computation.
	// For now we set it 20.
	var targetClearChance bool = false
	var searchSize int = 20 // =< 106(=256-30-1-119) is preferred
	lastDets := make([]float64, searchSize)
	var beginLastInterval int = 256 - dim - searchSize
	for j := 0; j < searchSize; j++ {
		i := beginLastInterval + j
		submatrix = submatCopy(matrix, i, dim)
		det := mat.Det(submatrix)
		detInt := int64(det)
		detBig := big.NewInt(detInt)
		lastDets[j] = det
		if detBig.Cmp(target) >= 0 {
			targetClearChance = true
		}
	}
	if !targetClearChance {
		return ret, nonZerosCount
	}
	for i := 1; i < beginLastInterval; i++ {
		submatrix = submatCopy(matrix, i, dim)
		det := mat.Det(submatrix)
		// check number of submatrices whose determinant is larger than 0
		detInt := int64(det)
		detBig := big.NewInt(detInt)		
		if detBig.Cmp(big.NewInt(0)) > 0 {
			nonZerosCount++
		}
		// already meet the requirement, just stop and return
		if nonZerosCount >= nonZeroCountTarget {
			return det, nonZerosCount
		}
		// at this point, even all left are ok, the total is still smaller than target, just stop!
		if nonZerosCount+(256-i-dim) < nonZeroCountTarget {
			return det, nonZerosCount
		}
	}
	for i := beginLastInterval; i < 256-dim; i++ {
		det := lastDets[i-beginLastInterval]
		// check number of det whose det is larger than 0
		if det > 0 {
			nonZerosCount++
		}
		// already meet the requirement, just stop and return
		if nonZerosCount >= nonZeroCountTarget {
			return det, nonZerosCount
		}
		// at this point, even all left are ok, the total is still smaller than target, just stop!
		if nonZerosCount+(256-i-dim) < nonZeroCountTarget {
			return det, nonZerosCount
		}
	}
	return ret, nonZerosCount
}

// matrix is dim x 256
// calDetmLoop will loop from 0 to 256 - dim
func calDetmLoopForVerification(matrix *mat.Dense, dim int, log *log.SeeleLog) (float64, int) {
	var ret = float64(0)
	var nonZerosCount = int(0)
	nonZeroCountTarget := getNonZeroCountTarget(dim)
	submatrix := mat.NewDense(dim, dim, nil)
	for i := 1; i < 256-dim; i++ {
		submatrix = submatCopy(matrix, i, dim)
		det := mat.Det(submatrix)
		// check number of det whose det is larger than 0
		detInt := int64(det)
		detBig := big.NewInt(detInt)
		if detBig.Cmp(big.NewInt(0)) > 0 {
			nonZerosCount++
		}
		// already meet the requirement, just stop and return
		if nonZerosCount >= nonZeroCountTarget {
			return det, nonZerosCount
		}
		// at this point, even all left are ok, the total is still smaller than target, just stop!
		if nonZerosCount+(256-i-dim) < nonZeroCountTarget {
			return det, nonZerosCount
		}
	}
	return ret, nonZerosCount
}

func submatCopy(matrix *mat.Dense, beginCol int, dim int) *mat.Dense {
	submatrix := mat.NewDense(dim, dim, nil)
	for i := 0; i < dim; i++ {
		col := mat.Col(nil, beginCol+i, matrix)
		submatrix.SetCol(i, col)
	}
	return submatrix
}

func submatCopyByRow(headerTarget *types.BlockHeader, matrix *mat.Dense, beginRow int, dim int, nonce uint64) *mat.Dense {

	header := headerTarget.Clone()
	submatrix := mat.NewDense(dim, 256, nil)
	for i := 0; i < dim-1; i++ {
		row := mat.Row(nil, beginRow+i, matrix)
		submatrix.SetRow(i, row)
	}
	header.Witness = []byte(strconv.FormatUint(nonce, 10))
	header.SecondWitness = []byte{}
	hash := header.Hash()
	col, isLeg := getBinaryArray(hash.String())
	if isLeg == false {
		return nil
	}
	submatrix.SetRow(dim-1, col)
	return submatrix
}

func getBinaryArray(hash string) ([]float64, bool) {
	binmap := map[int32][]float64{
		48:  {0, 0, 0, 0}, //0
		49:  {0, 0, 0, 1}, //1
		50:  {0, 0, 1, 0}, //2
		51:  {0, 0, 1, 1}, //3
		52:  {0, 1, 0, 0}, //4
		53:  {0, 1, 0, 1}, //5
		54:  {0, 1, 1, 0}, //6
		55:  {0, 1, 1, 1}, //7
		56:  {1, 0, 0, 0}, //8
		57:  {1, 0, 0, 1}, //9
		97:  {1, 0, 1, 0}, //a
		98:  {1, 0, 1, 1}, //b
		99:  {1, 1, 0, 0}, //c
		100: {1, 1, 0, 1}, //d
		101: {1, 1, 1, 0}, //e
		102: {1, 1, 1, 1}, //f
	}
	bits := make([]float64, 0)
	if !strings.HasPrefix(hash, "0x") {
		return bits, false
	}
	for _, c := range strings.TrimPrefix(hash, "0x") {
		bits = append(bits, binmap[c]...)
	}
	return bits, true
}
