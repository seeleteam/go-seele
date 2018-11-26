/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package consensus

import (
	"math/big"
	"sync"
	"sync/atomic"
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
	rewards int64
}

// all rewards of 3 hundred million
func Test_Rewards_10_Years(t *testing.T) {
	// three hundred million
	RewardNum := big.NewFloat(300000000)
	blockNum := blockNumberPerEra * 10
	var info reward
	threads := uint64(100)
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
			end = blockNum + 1
		} else {
			end = (i + 1) * per
		}

		go add(start, end, &info)
	}

	info.wg.Wait()

	shardRewards := big.NewInt(info.rewards)
	shardRewards.Div(shardRewards, common.SeeleToFan)
	allRewards := shardRewards.Mul(shardRewards, big.NewInt(int64(common.ShardCount))).Int64()
	FRewards := big.NewFloat(float64(allRewards))

	high := big.NewFloat(1)
	low := big.NewFloat(1)
	high.Mul(RewardNum, big.NewFloat(1.1))
	low.Mul(RewardNum, big.NewFloat(0.9))

	assert.Equal(t, 1, high.Cmp(FRewards))

	assert.Equal(t, -1, low.Cmp(FRewards))
}

func add(start, end uint64, info *reward) {
	defer info.wg.Done()

	count := int64(0)
	for i := start; i < end; i++ {
		reward := GetReward(i).Int64()
		count += reward
	}

	atomic.AddInt64(&info.rewards, count)
}
