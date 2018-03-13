/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"sync"
	"sync/atomic"

	"github.com/seeleteam/go-seele/consensus/pow"
)

type CPUEngine struct {
	mutex sync.Mutex

	taskChan    chan *Task
	stopChan    chan struct{}
	retChan     chan<- *Result
	currentChan chan struct{}

	consensus pow.Worker

	mining int32
}

func NewCPUEngine(consensus pow.Worker) *CPUEngine {
	engine := &CPUEngine{
		consensus: consensus,
		taskChan:  make(chan *Task, 1),
		stopChan:  make(chan struct{}, 1),
	}

	return engine
}

func (eng *CPUEngine) Task() chan<- *Task {
	return eng.taskChan
}

func (eng *CPUEngine) SetRetChan(ch chan<- *Result) {
	eng.retChan = ch
}

func (eng *CPUEngine) Start() {
	if atomic.LoadInt32(&eng.mining) == 1 {
		return
	}

	atomic.StoreInt32(&eng.mining, 1)
	go eng.doTask()
}

func (eng *CPUEngine) Stop() {
	if atomic.LoadInt32(&eng.mining) == 0 {
		return
	}

	eng.stopChan <- struct{}{}

	// clear all work in taskChan
out:
	for {
		select {
		case <-eng.taskChan:
		default:
			break out
		}
	}
}

func (eng *CPUEngine) doTask() {
out:
	for {
		select {
		case work := <-eng.taskChan:
			eng.mutex.Lock()
			if eng.currentChan != nil {
				close(eng.currentChan) // close current worker if exist
			}
			eng.currentChan = make(chan struct{})
			go eng.mine(work, eng.currentChan) // start mining
			eng.mutex.Unlock()
		case <-eng.stopChan:
			eng.mutex.Lock()
			if eng.currentChan != nil {
				close(eng.currentChan)
				eng.currentChan = nil
			}
			eng.mutex.Unlock()
			break out
		}
	}
}

func (eng *CPUEngine) mine(work *Task, stop chan<- struct{}) {
	// call pow
	// but the pow impl of consensus/pow is not enough
}
