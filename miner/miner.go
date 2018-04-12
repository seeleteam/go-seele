/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"math/big"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/seele"
)

// Miner defines base elements of miner
type Miner struct {
	coinbase common.Address
	mining   int32
	canStart int32

	stopChan chan struct{}
	current  *Task
	recv     chan *Result

	seele *seele.SeeleService
	log   *log.SeeleLog

	isFirstDownloader int32
}

// NewMiner construct a miner, return a Miner instance
func NewMiner(addr common.Address, seele *seele.SeeleService, log *log.SeeleLog) *Miner {
	miner := &Miner{
		coinbase: addr,
		canStart: 1,
		seele:    seele,
		stopChan: make(chan struct{}, 1),
		recv:     make(chan *Result, 1),
		log:      log,
		isFirstDownloader: 1,
	}

	event.BlockDownloaderEventManager.AddAsyncListener(miner.downloadEventCallback)
	event.TransactionInsertedEventManager.AddAsyncListener(miner.newTxCallback)

	return miner
}

// Start function is used to start miner
func (miner *Miner) Start() bool {
	if atomic.LoadInt32(&miner.mining) == 1 {
		miner.log.Info("Miner is running")
		return true
	}

	if atomic.LoadInt32(&miner.canStart) == 0 {
		miner.log.Info("Can not start miner when syncing")
		return false
	}

	atomic.StoreInt32(&miner.mining, 1)

	go miner.waitBlock()
	miner.prepareNewBlock() // try to prepare the first block

	return true
}

// Stop function is used to stop miner
func (miner *Miner) Stop() {
	atomic.StoreInt32(&miner.mining, 0)
	miner.stopChan <- struct{}{}
}

func (miner *Miner) Close() {
	close(miner.stopChan)
	close(miner.recv)
}

// IsMining returns true if miner is started, return false if not
func (miner *Miner) IsMining() bool {
	return atomic.LoadInt32(&miner.mining) == 1
}

func (miner *Miner) downloadEventCallback(e event.Event) {
	if atomic.LoadInt32(&miner.isFirstDownloader) == 0 {
		return
	}

	eventType := e.(int)
	switch eventType {
	case event.DownloaderStartEvent:
		atomic.StoreInt32(&miner.canStart, 0)
		if miner.IsMining() {
			miner.Stop()
		}
	case event.DownloaderDoneEvent, event.DownloaderFailedEvent:
		atomic.StoreInt32(&miner.isFirstDownloader, 0)
		atomic.StoreInt32(&miner.canStart, 1)
		miner.Start()
	}
}

func (miner *Miner) newTxCallback(e event.Event) {
	miner.log.Debug("got new tx event")
	// if not mining, start mining
	if atomic.LoadInt32(&miner.canStart) == 1 && atomic.CompareAndSwapInt32(&miner.mining, 0, 1) {
		miner.prepareNewBlock()
	}
}

func (miner *Miner) waitBlock() {
out:
	for {
		select {
		case result := <-miner.recv:
			if result == nil || result.task != miner.current {
				continue
			}

			ret := miner.saveBlock(result)
			if ret != nil {
				miner.log.Error("saveBlock failed, cause for %s", ret)
				continue
			}

			miner.log.Info("found a new mined block, notify to p2p")
			event.BlockMinedEventManager.Fire(result.block) // notify p2p to broadcast block
			atomic.StoreInt32(&miner.mining, 0)

			// loop mining after mining complete
			miner.newTxCallback(event.EmptyEvent)
		case <-miner.stopChan:
			break out
		}
	}
}

func (miner *Miner) prepareNewBlock() {
	miner.log.Debug("start mining new block")

	timestamp := time.Now().Unix()
	parent, stateDB := miner.seele.BlockChain().CurrentBlock()

	if parent.Header.CreateTimestamp.Cmp(new(big.Int).SetInt64(timestamp)) >= 0 {
		timestamp = parent.Header.CreateTimestamp.Int64() + 1
	}

	// this will ensure we're not going off too far in the future
	if now := time.Now().Unix(); timestamp > now+1 {
		wait := time.Duration(timestamp-now) * time.Second
		miner.log.Info("Mining too far in the future, wait for %s", wait)
		time.Sleep(wait)
	}

	height := parent.Header.Height
	header := &types.BlockHeader{
		PreviousBlockHash: parent.HeaderHash,
		Creator:           miner.coinbase,
		Height:            height + 1,
		CreateTimestamp:   big.NewInt(timestamp),
		Difficulty:        big.NewInt(10000000), //TODO find a way to decide difficulty
	}

	miner.current = &Task{
		header:    header,
		createdAt: time.Now(),
	}

	txs := miner.seele.TxPool().GetProcessableTransactions()
	txSlice := make([]*types.Transaction, 0)
	for _, value := range txs {
		txSlice = append(txSlice, value...)
	}

	err := miner.current.applyTransactions(miner.seele, stateDB.GetCopy(), txSlice, miner.log)
	if err != nil {
		miner.log.Warn(err.Error())
		atomic.StoreInt32(&miner.mining, 0)
		return
	}

	miner.log.Info("commit a new task to engine, height=%d", header.Height)
	miner.commitTask(miner.current)
}

func (miner *Miner) saveBlock(result *Result) error {
	ret := miner.seele.BlockChain().WriteBlock(result.block)
	return ret
}

func (miner *Miner) commitTask(task *Task) {
	if atomic.LoadInt32(&miner.mining) != 1 {
		return
	}

	go StartMining(task, rand.Uint64(), miner.recv, miner.stopChan, miner.log)
}
