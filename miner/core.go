/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"math/big"
	"sync/atomic"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner/pow"
)

// StartMining starts calculating the nonce for the block.
// seed is the random start value for the nonce
// result represents the founded nonce will be set in the result block
// abort is a channel by closing which you can stop mining
func StartMining(task *Task, seed uint64, min uint64, max uint64, result chan<- *Result, abort <-chan struct{}, isNonceFound *int32, log *log.SeeleLog) {
	block := task.generateBlock()

	var nonce = seed
	var hashInt big.Int
	target := pow.GetMiningTarget(block.Header.Difficulty)

miner:
	for {
		select {
		case <-abort:
			logAbort(log)
			break miner

		default:
			if *isNonceFound != 0 {
				log.Info("nonce is found")
				break miner
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
					log.Info("nonce finding succeeded")
				}

				break miner
			}

			// outage
			if nonce == max {
				nonce = min
			}
			if nonce == seed-1 {
				select {
				case <-abort:
					logAbort(log)
				case result <- nil:
					log.Info("nonce finding outage")
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
