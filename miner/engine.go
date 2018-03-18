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

// CPUEngine is a mine engine used to find a good nonce for block
type CPUEngine struct {
	mutex sync.Mutex

	taskChan    chan *Task
	stopChan    chan struct{}
	retChan     chan<- *Result
	currentChan chan struct{}

	consensus pow.Worker

	mining int32
}

// NewCPUEngine is constructor of CPUEngine
func NewCPUEngine(consensus pow.Worker) *CPUEngine {
	engine := &CPUEngine{
		consensus: consensus,
		taskChan:  make(chan *Task, 1),
		stopChan:  make(chan struct{}, 1),
	}

	return engine
}

// Task return taskChan
func (eng *CPUEngine) Task() chan<- *Task {
	return eng.taskChan
}

// SetRetChan set retChan
func (eng *CPUEngine) SetRetChan(ch chan<- *Result) {
	eng.retChan = ch
}

// Start function used to start engine
func (eng *CPUEngine) Start() {
	if !atomic.CompareAndSwapInt32(&eng.mining, 0, 1) {
		return
	}

	go eng.doTask()
}

// Stop function used to stop engine
func (eng *CPUEngine) Stop() {
	if !atomic.CompareAndSwapInt32(&eng.mining, 1, 0) {
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
