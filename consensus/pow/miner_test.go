/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

import (
	"math/big"
	"testing"

	"sync"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/core/types"
)

func getBlock(difficult int64) *types.Block {
	return &types.Block{
		Header: &types.BlockHeader{
			Difficulty: big.NewInt(difficult),
		},
	}
}

func Test_Worker(t *testing.T) {
	block := getBlock(10)

	result := make(chan *types.Block)
	abort := make(chan interface{})
	go Mine(block, result, abort)

	select {
	case <-result:
		target := new(big.Int).Div(maxUint256, block.Header.Difficulty)
		hash := block.Header.Hash()
		var hashInt big.Int
		hashInt.SetBytes(hash.Bytes())
		assert.Equal(t, hashInt.Cmp(target) <= 0, true)
	}
}

func Test_WorkerStop(t *testing.T) {
	block := getBlock(20)

	result := make(chan *types.Block)
	abort := make(chan interface{})

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		Mine(block, result, abort)
		wg.Done()
	}()

	close(abort)

	wg.Wait()
}
