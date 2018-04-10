/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"math"
	"math/big"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner/pow"
)

// StartMining start calculate nonce for the block.
// seed random start value for nonce
// result found nonce will be set in the result block
// abort you could stop it by close(abort)
func StartMining(task *Task, seed uint64, result chan<- *Result, abort <-chan struct{}, log *log.SeeleLog) {
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
					log.Info("nonce found succeed")
				}

				break miner
			}

			// outage
			if nonce == math.MaxUint64 {
				select {
				case <-abort:
					logAbort(log)
				case result <- nil:
					log.Info("nonce found outage")
				}

				break miner
			}

			nonce++
		}
	}
}

func logAbort(log *log.SeeleLog) {
	log.Info("nonce found abort")
}
