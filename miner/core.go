/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"math"
	"math/big"

	"github.com/seeleteam/go-seele/log"
)

var (
	// maxUint256 is a big integer representing 2^256
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
)

// StartMining start calculate nonce for the block.
// seed random start value for nonce
// result found nonce will be set in the result block
// abort you could stop it by close(abort)
func StartMining(task *Task, seed uint64, result chan<- *Result, abort <-chan struct{}, log *log.SeeleLog) {
	block := task.generateBlock()

	var nonce = seed
	var hashInt big.Int
	target := new(big.Int).Div(maxUint256, block.Header.Difficulty)

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
			if hashInt.Cmp(target) <= 0 {
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
