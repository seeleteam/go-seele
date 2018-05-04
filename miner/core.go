/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"math/big"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner/pow"
)

// StartMining starts calculating the nonce for the block.
// seed is the random start value for the nonce
// result represents the founded nonce will be set in the result block
// abort is a channel by closing which you can stop mining
func StartMining(task *Task, seed uint64, max uint64, result chan<- *Result, abort <-chan struct{}, log *log.SeeleLog) {
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
					log.Info("nonce finding succeeded")
				}

				break miner
			}

			// outage
			if nonce == max {
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
