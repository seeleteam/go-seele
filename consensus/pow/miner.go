/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

import (
	"math"
	"math/big"

	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
)

var (
	// maxUint256 is a big integer representing 2^256
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

	logger = log.GetLogger("miner", true)
)

// Mine start calculate nonce for the block.
// result found nonce will be set in the result block
// abort you could stop it by close(abort)
func Mine(block *types.Block, result chan<- *types.Block, abort <-chan interface{}) {
	var nonce uint64
	var hashInt big.Int
	target := new(big.Int).Div(maxUint256, block.Header.Difficulty)

miner:
	for {
		select {
		case <-abort:
			exit()
			break miner

		default:
			block.Header.Nonce = nonce
			hash := block.Header.Hash()
			hashInt.SetBytes(hash.Bytes())
			if hashInt.Cmp(target) <= 0 {
				select {
				case <-abort:
					exit()
				case result <- block:
					logger.Info("nonce found succeed")
				}

				break miner
			}

			if nonce == math.MaxUint64 {
				select {
				case <-abort:
					exit()
				case result <- nil:
					logger.Info("nonce found outage")
				}

				break miner
			}

			nonce++
		}
	}
}

func exit() {
	logger.Info("nonce found abort")
}
