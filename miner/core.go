/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"math/big"
	"sync/atomic"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner/pow"
)

// StartMining starts calculating the nonce for the block.
// seed is the random start value for the nonce
// min is the min number for the nonce per thread
// max is the max number for the nonce per thread
// result represents the founded nonce will be set in the result block
// abort is a channel by closing which you can stop mining
// isNonceFound is a flag to mark nonce is found by other threads
// hashrate is the average hashrate of miner
func StartMining(task *Task, seed uint64, min uint64, max uint64, result chan<- *Result, abort <-chan struct{}, isNonceFound *int32, hashrate metrics.Meter, log *log.SeeleLog) {
	block := task.generateBlock()

	var nonce = seed
	var hashInt big.Int
	var caltimes = int64(0)
	target := pow.GetMiningTarget(block.Header.Difficulty)

miner:
	for {
		select {
		case <-abort:
			logAbort(log)
			hashrate.Mark(caltimes)
			break miner

		default:
			if atomic.LoadInt32(isNonceFound) != 0 {
				log.Info("exit mining as nonce is found by other threads")
				break miner
			}

			caltimes++
			if caltimes == 0x7FFF {
				hashrate.Mark(caltimes)
				caltimes = 0
			}

			block.Header.Nonce = nonce
			hash := block.Header.Hash()
			hashInt.SetBytes(hash.Bytes())

			// found
			if hashInt.Cmp(target) <= 0 {
				block.HeaderHash = hash
				found := &Result{
					task:  task,
					block: block,
				}

				select {
				case <-abort:
					logAbort(log)
				case result <- found:
					atomic.StoreInt32(isNonceFound, 1)
					log.Debug("nonce finding succeeded")
				}

				break miner
			}

			// when nonce reached max, nonce traverses in [min, seed-1]
			if nonce == max {
				nonce = min
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

// logAbort logs the info that nonce finding is aborted
func logAbort(log *log.SeeleLog) {
	log.Info("nonce finding aborted")
}
