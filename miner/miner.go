/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/seele"
)

// Task is a mining work for engine, it contains block header, transactions, and transaction receipts.
type Task struct {
	block  *types.Block
	header *types.BlockHeader
	txs    []*types.Transaction
	//receipts   []*types.Receipt

	//state      *state.StateDB

	createdAt time.Time
}

// Result if struct of mined result by engine, it contains the raw task and mined block.
type Result struct {
	Task  *Task
	Block *types.Block // mined block, with good nonce
}

// Engine is interface of engine
type Engine interface {
	TaskChan() chan<- *Task
	SetRetChan(chan<- *Result)
	Start()
	Stop()
}

// Miner defines base elements of miner
type Miner struct {
	mutex sync.Mutex

	coinbase common.Address
	mining   int32
	canStart int32

	seele seele.SeeleService

	engines map[Engine]struct{}

	txChan   chan *types.Transaction
	headChan chan *types.BlockHeader
	stopChan chan struct{}

	current *Task
	recv    chan *Result

	log		*log.SeeleLog
}

// NewMiner construct a miner, return a Miner instance
func NewMiner(addr common.Address, seele seele.SeeleService, log *log.SeeleLog) *Miner {
	miner := &Miner{
		coinbase: addr,
		canStart: 1,
		seele:    seele,
		engines:  make(map[Engine]struct{}),
		stopChan: make(chan struct{}, 1),
		log:	  log,
	}

	event.BlockDownloaderEventManager.AddListener(miner.downloadEventCallback)
	event.TransactionInsertedEventManager.AddListener(miner.newTxCallback)
	event.BlockInsertedEventManager.AddListener(miner.newBlockCallback)

	return miner
}

// Start function is used to start miner
func (miner *Miner) Start() bool {
	miner.mutex.Lock()
	defer miner.mutex.Unlock()

	if atomic.LoadInt32(&miner.mining) == 1 {
		miner.log.Info("Miner is running")
		return true
	}

	if atomic.LoadInt32(&miner.canStart) == 0 {
		miner.log.Info("Can not start miner when syncing")
		return false
	}

	atomic.StoreInt32(&miner.mining, 1)

	// start engine
	for engine := range miner.engines {
		engine.Start()
	}

	go miner.updateBlock()
	go miner.waitBlock()

	miner.prepareNewBlock() // try to prepare the first block

	return true
}

// Stop function is used to stop miner
func (miner *Miner) Stop() {
	miner.mutex.Lock()
	defer miner.mutex.Unlock()

	atomic.StoreInt32(&miner.mining, 0)
	close(miner.stopChan)

	for engine := range miner.engines {
		engine.Stop()
	}
}

// IsMining returns true if miner is started, return false if not
func (miner *Miner) IsMining() bool {
	return atomic.LoadInt32(&miner.mining) == 1
}

// RegisterEngine adds one engine to miner
func (miner *Miner) RegisterEngine(engine Engine) {
	miner.mutex.Lock()
	defer miner.mutex.Unlock()

	miner.engines[engine] = struct{}{}
	engine.SetRetChan(miner.recv)

	if miner.IsMining() {
		engine.Start()
	}
}

// UnregisterEngine delete one engine
func (miner *Miner) UnregisterEngine(engine Engine) {
	miner.mutex.Lock()
	defer miner.mutex.Unlock()

	delete(miner.engines, engine)
	engine.Stop()
}

func (miner *Miner) downloadEventCallback(e event.Event) {
	eventType := e.(int)
	switch eventType {
	case event.DownloaderStartEvent:
		atomic.StoreInt32(&miner.canStart, 0)
		if miner.IsMining() {
			miner.Stop()
		}
	case event.DownloaderDoneEvent, event.DownloaderFailedEvent:
		atomic.StoreInt32(&miner.canStart, 1)
		miner.Start()
	}
}

func (miner *Miner) newTxCallback(e event.Event) {
	tx := e.(*types.Transaction)
	miner.txChan <- tx
}

func (miner *Miner) newBlockCallback(e event.Event) {
	block := e.(*types.BlockHeader)
	miner.headChan <- block
}

func (miner *Miner) updateBlock() {
out:
	for {
		select {
		case <-miner.txChan:
			// TODO:
		case <-miner.headChan:
			miner.prepareNewBlock()
		case <-miner.stopChan:
			break out
		}
	}
}

func (miner *Miner) waitBlock() {
out:
	for {
		select {
		case result := <-miner.recv:
			if result == nil {
				continue
			}

			ret := miner.saveBlock(result)
			if ret != nil {
				miner.log.Error("saveBlock failed, cause for %s", ret)
				continue
			}

			miner.log.Info("found a new mined block, notify to p2p")
			event.BlockMinedEventManager.Fire(result.Block) // notify p2p to broadcast block

			miner.prepareNewBlock() // start a new block if save one newest block successfully
		case <-miner.stopChan:
			break out
		}
	}
}

func (miner *Miner) prepareNewBlock() {
	miner.mutex.Lock()
	defer miner.mutex.Unlock()

	tstart := time.Now()
	parent := miner.seele.BlockChain().CurrentBlock()

	tstamp := tstart.Unix()
	if parent.Header.CreateTimestamp.Cmp(new(big.Int).SetInt64(tstamp)) >= 0 {
		tstamp = parent.Header.CreateTimestamp.Int64() + 1
	}
	// this will ensure we're not going off too far in the future
	if now := time.Now().Unix(); tstamp > now + 1 {
		wait := time.Duration(tstamp-now) * time.Second
		//log.Info("Mining too far in the future", "wait", common.PrettyDuration(wait))
		time.Sleep(wait)
	}

	height := parent.Header.Height
	header := &types.BlockHeader{
		PreviousBlockHash: parent.HeaderHash,
		Creator:           miner.coinbase,
		Height:            height + 1,
		CreateTimestamp:   big.NewInt(tstamp),
	}

	miner.current = &Task{
		header:    header,
		createdAt: time.Now(),
	}

	pendingTx, err := miner.seele.TxPool().Pending()
	if err != nil {
		miner.log.Error("get pending txs failed, cause for %s", err)
		return
	}
	miner.current.applyTransactions(miner.seele, miner.coinbase, pendingTx, miner.log)

	miner.current.block = types.NewBlock(miner.current.header, miner.current.txs)

	miner.log.Info("commit a new task to engine, height=%d", header.Height)
	miner.commitTask(miner.current)
}

func (miner *Miner) saveBlock(result *Result) error {
	// call blockchain to write
	ret := miner.seele.BlockChain().WriteBlock(result.Block)
	// TODO: write recepit&state
	return ret
}

func (miner *Miner) commitTask(task *Task) {
	if atomic.LoadInt32(&miner.mining) != 1 {
		return
	}

	// notify engine
	for engine := range miner.engines {
		if ch := engine.TaskChan(); ch != nil {
			ch <- task
		}
	}
}

func (task *Task) applyTransactions(seele seele.SeeleService, coinbase common.Address, txs []*types.Transaction, log *log.SeeleLog) {
	for _, tx := range txs {
		// execute tx
		err := seele.ApplyTransaction(coinbase, tx)
		if err != nil {
			log.Error("exec tx failed, cause for %s", err)
			continue
		}

		task.txs = append(task.txs, tx)
	}
}
