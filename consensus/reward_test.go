/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package consensus

import (
	"testing"

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
