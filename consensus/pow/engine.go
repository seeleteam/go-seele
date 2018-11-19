/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

import (
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/utils"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/rpc"
)

var (
	// maxUint256 is a big integer representing 2^256
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
)

// Engine provides the consensus operations based on POW.
type Engine struct {
	threads  int
	log      *log.SeeleLog
	hashrate metrics.Meter
}

func NewEngine(threads int) *Engine {
	return &Engine{
		threads:  threads,
		log:      log.GetLogger("pow_engine"),
		hashrate: metrics.NewMeter(),
	}
}

func (engine *Engine) SetThreads(threads int) {
	if threads <= 0 {
		engine.threads = runtime.NumCPU()
	} else {
		engine.threads = threads
	}
}

func (engine *Engine) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{
		{
			Namespace: "miner",
			Version:   "1.0",
			Service:   &API{engine},
			Public:    true,
		},
	}
}

// ValidateHeader validates the specified header and returns error if validation failed.
func (engine *Engine) VerifyHeader(reader consensus.ChainReader, header *types.BlockHeader) error {
	parent := reader.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return consensus.ErrBlockInvalidParentHash
	}

	if err := utils.VerifyHeaderCommon(header, parent); err != nil {
		return err
	}

	if err := verifyTarget(header); err != nil {
		return err
	}

	return nil
}

func (engine *Engine) Prepare(reader consensus.ChainReader, header *types.BlockHeader) error {
	parent := reader.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return consensus.ErrBlockInvalidParentHash
	}

	header.Difficulty = utils.GetDifficult(header.CreateTimestamp.Uint64(), parent)

	return nil
}

func (engine *Engine) Seal(reader consensus.ChainReader, block *types.Block, stop <-chan struct{}, results chan<- *types.Block) error {
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
			StartMining(block, tseed, tmin, tmax, results, stop, &isNonceFound, once, engine.hashrate, engine.log)
		}(tSeed, min, max)
	}

	return nil
}

func verifyTarget(header *types.BlockHeader) error {
	headerHash := header.Hash()
	var hashInt big.Int
	hashInt.SetBytes(headerHash.Bytes())

	target := getMiningTarget(header.Difficulty)

	if hashInt.Cmp(target) > 0 {
		return consensus.ErrBlockNonceInvalid
	}

	return nil
}

// getMiningTarget returns the mining target for the specified difficulty.
func getMiningTarget(difficulty *big.Int) *big.Int {
	return new(big.Int).Div(maxUint256, difficulty)
}
