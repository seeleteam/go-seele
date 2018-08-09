/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner/pow"
)

var (
	// ErrMinerIsRunning is returned when miner is running
	ErrMinerIsRunning = errors.New("miner is running")

	// ErrMinerIsStopped is returned when miner is stopped
	ErrMinerIsStopped = errors.New("miner is stopped")

	// ErrNodeIsSyncing is returned when the node is syncing
	ErrNodeIsSyncing = errors.New("can not start miner when syncing")
)

// SeeleBackend wraps all methods required for minier.
type SeeleBackend interface {
	TxPool() *core.TransactionPool
	BlockChain() *core.Blockchain
}

// Miner defines base elements of miner
type Miner struct {
	coinbase common.Address
	mining   int32
	canStart int32
	stopped  int32

	wg       sync.WaitGroup
	stopChan chan struct{}
	current  *Task
	recv     chan *Result

	seele SeeleBackend
	log   *log.SeeleLog

	isFirstDownloader int32

	threads              int
	isFirstBlockPrepared int32
	hashrate             metrics.Meter // Meter tracking the average hashrate
}

// NewMiner constructs and returns a miner instance
func NewMiner(addr common.Address, seele SeeleBackend) *Miner {
	miner := &Miner{
		coinbase:             addr,
		canStart:             1,
		stopped:              0,
		seele:                seele,
		wg:                   sync.WaitGroup{},
		recv:                 make(chan *Result, 1),
		log:                  log.GetLogger("miner"),
		isFirstDownloader:    1,
		isFirstBlockPrepared: 0,
		threads:              1,
		hashrate:             metrics.NewMeter(),
	}

	event.BlockDownloaderEventManager.AddAsyncListener(miner.downloaderEventCallback)
	event.TransactionInsertedEventManager.AddAsyncListener(miner.newTxCallback)

	return miner
}

// GetCoinbase returns the coinbase.
func (miner *Miner) GetCoinbase() common.Address {
	return miner.coinbase
}

// SetThreads sets the number of mining threads.
func (miner *Miner) SetThreads(threads uint) {
	if threads == 0 {
		miner.threads = runtime.NumCPU()
		return
	}

	miner.threads = int(threads)
}

// GetThreads gets the number of mining threads.
func (miner *Miner) GetThreads() int {
	return miner.threads
}

// SetCoinbase set the coinbase.
func (miner *Miner) SetCoinbase(coinbase common.Address) {
	miner.coinbase = coinbase
}

// Start is used to start the miner
func (miner *Miner) Start() error {
	if atomic.LoadInt32(&miner.mining) == 1 {
		miner.log.Info("Miner is running")
		return ErrMinerIsRunning
	}

	if atomic.LoadInt32(&miner.canStart) == 0 {
		miner.log.Info("Can not start miner when syncing")
		return ErrNodeIsSyncing
	}

	// CAS to ensure only 1 mining goroutine.
	if !atomic.CompareAndSwapInt32(&miner.mining, 0, 1) {
		miner.log.Info("Another goroutine has already started to mine")
		return nil
	}

	miner.log.Info("miner start with %d threads", miner.threads)
	miner.stopChan = make(chan struct{})

	if err := miner.prepareNewBlock(); err != nil { // try to prepare the first block
		miner.log.Warn(err.Error())
		atomic.StoreInt32(&miner.mining, 0)

		return err
	}

	atomic.StoreInt32(&miner.stopped, 0)
	go miner.waitBlock()

	miner.log.Info("Miner is started.")

	return nil
}

// Stop is used to stop the miner
func (miner *Miner) Stop() {
	// set stopped to 1 to prevent restart
	atomic.StoreInt32(&miner.stopped, 1)
	miner.stopMining()
}

func (miner *Miner) stopMining() {
	if !atomic.CompareAndSwapInt32(&miner.mining, 1, 0) {
		return
	}

	// notify all threads to terminate
	if miner.stopChan != nil {
		close(miner.stopChan)
		miner.stopChan = nil
	}

	// wait for all threads to terminate
	miner.wg.Wait()

	miner.log.Info("Miner is stopped.")
}

// Close closes the miner.
func (miner *Miner) Close() {
	if miner.stopChan != nil {
		close(miner.stopChan)
		miner.stopChan = nil
	}

	if miner.recv != nil {
		close(miner.recv)
		miner.recv = nil
	}
}

// IsMining returns true if the miner is started, otherwise false
func (miner *Miner) IsMining() bool {
	return atomic.LoadInt32(&miner.mining) == 1
}

// downloaderEventCallback handles events which indicate the downloader state
func (miner *Miner) downloaderEventCallback(e event.Event) {
	if atomic.LoadInt32(&miner.isFirstDownloader) == 0 {
		return
	}

	switch e.(int) {
	case event.DownloaderStartEvent:
		miner.log.Info("got download start event, stop miner")
		atomic.StoreInt32(&miner.canStart, 0)
		if miner.IsMining() {
			miner.stopMining()
		}
	case event.DownloaderDoneEvent, event.DownloaderFailedEvent:
		atomic.StoreInt32(&miner.canStart, 1)
		atomic.StoreInt32(&miner.isFirstDownloader, 0)

		if atomic.LoadInt32(&miner.stopped) == 0 {
			miner.log.Info("got download end event, start miner")
			miner.Start()
		}
	}
}

// newTxCallback handles the new tx event
func (miner *Miner) newTxCallback(e event.Event) {
	if common.PrintExplosionLog {
		miner.log.Debug("got the new tx event")
	}

	// if not mining, start mining
	if atomic.LoadInt32(&miner.stopped) == 0 && atomic.LoadInt32(&miner.canStart) == 1 && atomic.CompareAndSwapInt32(&miner.mining, 0, 1) {
		if err := miner.prepareNewBlock(); err != nil {
			miner.log.Warn(err.Error())
			atomic.StoreInt32(&miner.mining, 0)
		}
	}
}

// waitBlock waits for blocks to be mined continuously
func (miner *Miner) waitBlock() {
out:
	for {
		select {
		case result := <-miner.recv:
			for {
				if result == nil || result.task != miner.current {
					break
				}

				miner.log.Info("found a new mined block, block height:%d, hash:%s", result.block.Header.Height, result.block.HeaderHash.ToHex())
				ret := miner.saveBlock(result)
				if ret != nil {
					miner.log.Error("failed to save the block, for %s", ret.Error())
					break
				}

				miner.log.Info("block and notify p2p saved successfully")
				event.BlockMinedEventManager.Fire(result.block) // notify p2p to broadcast the block
				break
			}

			atomic.StoreInt32(&miner.mining, 0)
			// loop mining after mining completed
			miner.newTxCallback(event.EmptyEvent)
		case <-miner.stopChan:
			break out
		}
	}
}

// prepareNewBlock prepares a new block to be mined
func (miner *Miner) prepareNewBlock() error {
	miner.log.Debug("starting mining the new block")

	timestamp := time.Now().Unix()
	parent, stateDB, err := miner.seele.BlockChain().GetCurrentInfo()
	if err != nil {
		return fmt.Errorf("failed to get current info, %s", err)
	}

	if parent.Header.CreateTimestamp.Cmp(new(big.Int).SetInt64(timestamp)) >= 0 {
		timestamp = parent.Header.CreateTimestamp.Int64() + 1
	}

	// this will ensure we're not going off too far in the future
	if now := time.Now().Unix(); timestamp > now+1 {
		wait := time.Duration(timestamp-now) * time.Second
		miner.log.Info("Mining too far in the future, waiting for %s", wait)
		time.Sleep(wait)
	}

	height := parent.Header.Height
	difficult := pow.GetDifficult(uint64(timestamp), parent.Header)
	header := &types.BlockHeader{
		PreviousBlockHash: parent.HeaderHash,
		Creator:           miner.coinbase,
		Height:            height + 1,
		CreateTimestamp:   big.NewInt(timestamp),
		Difficulty:        difficult,
	}

	miner.log.Debug("miner a block with coinbase %s", miner.coinbase.ToHex())
	miner.current = &Task{
		header:    header,
		createdAt: time.Now(),
		coinbase:  miner.coinbase,
	}

	err = miner.current.applyTransactions(miner.seele, stateDB, miner.log)
	if err != nil {
		return fmt.Errorf("failed to apply transaction %s", err)
	}

	miner.log.Info("committing a new task to engine, height:%d, difficult:%d", header.Height, header.Difficulty)
	miner.commitTask(miner.current)

	return nil
}

// saveBlock saves the block in the given result to the blockchain
func (miner *Miner) saveBlock(result *Result) error {
	ret := miner.seele.BlockChain().WriteBlock(result.block)
	return ret
}

// commitTask commits the given task to the miner
func (miner *Miner) commitTask(task *Task) {
	if atomic.LoadInt32(&miner.mining) != 1 {
		return
	}

	threads := miner.threads
	miner.log.Debug("miner threads num:%d", threads)

	var step uint64
	var seed uint64
	if threads != 0 {
		step = math.MaxUint64 / uint64(threads)
	}

	var isNonceFound int32
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
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

		miner.wg.Add(1)
		go func(tseed uint64, tmin uint64, tmax uint64) {
			defer miner.wg.Done()
			StartMining(task, tseed, tmin, tmax, miner.recv, miner.stopChan, &isNonceFound, miner.hashrate, miner.log)
		}(tSeed, min, max)
	}
}

// Hashrate returns the rate of the POW search invocations per second in the last minute.
func (miner *Miner) Hashrate() float64 {
	return miner.hashrate.Rate1()
}
