/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package consensus

import (
	"fmt"
	"math/big"
	"sync"
	"testing"

	"github.com/seeleteam/go-seele/common"

	"github.com/stretchr/testify/assert"
)

func Test_Reward(t *testing.T) {
	assert.Equal(t, GetReward(0), rewardTableCoin[0], "0")
	assert.Equal(t, GetReward(blockNumberPerEra-1), rewardTableCoin[0], "1")

	assert.Equal(t, GetReward(blockNumberPerEra), rewardTableCoin[1], "2")
	assert.Equal(t, GetReward(blockNumberPerEra*2-1), rewardTableCoin[1], "3")

	assert.Equal(t, GetReward(blockNumberPerEra*uint64(len(rewardTableCoin))-1), rewardTableCoin[len(rewardTableCoin)-1], "4")

	assert.Equal(t, GetReward(blockNumberPerEra*uint64(len(rewardTableCoin))), tailRewardCoin, "5")
}

type reward struct {
	wg      sync.WaitGroup
	lock    sync.Mutex
	rewards *big.Int
}

// all rewards of 3 hundred million
func Test_Rewards_15_365(t *testing.T) {
	seconds := uint64(3600 * 24 * 365 * 10)
	blockNum := seconds / 15
	var info reward
	info.rewards = big.NewInt(0)
	threads := uint64(1000)
	per := blockNum / threads

	var start uint64
	var end uint64

	for i := uint64(0); i < threads; i++ {
		info.wg.Add(1)

		if i == 0 {
			start = uint64(1)
		} else {
			start = i * per
		}

		if i == threads-1 {
			end = blockNum
		} else {
			end = (i + 1) * per
		}

		go add(start, end, &info)
	}

	info.wg.Wait()

	info.rewards.Div(info.rewards, common.SeeleToFan)
	fmt.Println("all rewards:", info.rewards)

}

func add(start, end uint64, info *reward) {
	defer info.wg.Done()

	count := big.NewInt(0)
	for i := start; i < end; i++ {
		reward := GetReward(i)
		count.Add(count, reward)
	}

	info.lock.Lock()
	info.rewards.Add(info.rewards, count)
	info.lock.Unlock()
}
