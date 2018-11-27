/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package consensus

import (
	"fmt"
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
	for i := 0; i < 10; i++ {
		reward := GetReward(uint64(i)*blockNumberPerEra + 1)
		rewardYear := new(big.Int).Mul(reward, new(big.Int).SetUint64(blockNumberPerEra))
		fmt.Println(rewardYear)
		sum = new(big.Int).Add(sum, rewardYear)
	}

	fmt.Println(sum)
	fmt.Println(targetReward)

	duration := new(big.Int).Div(targetReward, big.NewInt(10))
	assert.True(t, sum.Cmp(new(big.Int).Add(targetReward, duration)) < 0)
	assert.True(t, sum.Cmp(new(big.Int).Sub(targetReward, duration)) > 0)
}
