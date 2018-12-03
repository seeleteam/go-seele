/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package consensus

import (
	"math/big"
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

func Test_RewardTotal(t *testing.T) {
	targetReward := new(big.Int).Mul(new(big.Int).SetInt64(300000000), common.SeeleToFan)

	sum := big.NewInt(0)
	for i := uint64(0); i < 10*blockNumberPerEra; i++ {
		reward := GetReward(i)
		sum = new(big.Int).Add(sum, reward)
	}

	sum = new(big.Int).Mul(sum, big.NewInt(common.ShardCount))

	duration := new(big.Int).Div(targetReward, big.NewInt(100))
	assert.True(t, sum.Cmp(new(big.Int).Add(targetReward, duration)) < 0)
	assert.True(t, sum.Cmp(new(big.Int).Sub(targetReward, duration)) > 0)
}
