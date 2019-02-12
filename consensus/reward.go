/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package consensus

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
)

var (
	// rewardTable the reward value is per year. Which means the first value is for first year, second value is for second year, etc...
	rewardTableCoin []*big.Int

	// tailReward it is used when out of the reward table. we use a constant reward value.
	tailRewardCoin *big.Int

	// blockNumberPerEra block number per reward era. It is approximation of block number generated per year.
	blockNumberPerEra uint64 = 3150000
)

func init() {
	rewardTable := [...]float64{24, 16, 12, 10, 8, 8, 6, 6}
	tailReward := float64(6)

	rewardTableCoin = make([]*big.Int, len(rewardTable))
	for i, r := range rewardTable {
		rewardTableCoin[i] = convertSeeleToFan(r / common.ShardCount)
	}

	tailRewardCoin = convertSeeleToFan(tailReward / common.ShardCount)
}

func convertSeeleToFan(seele float64) *big.Int {
	unit := common.SeeleToFan.Int64()
	f := uint64(seele * float64(unit))

	return big.NewInt(0).SetUint64(f)
}

// GetReward get reward amount according to block height
func GetReward(blockHeight uint64) *big.Int {
	era := int(blockHeight / blockNumberPerEra)

	var result *big.Int
	if era < len(rewardTableCoin) {
		result = rewardTableCoin[era]
	} else if era == len(rewardTableCoin) {
		result = tailRewardCoin
	}

	if era > len(rewardTableCoin) {
		result = big.NewInt(0)
	}

	return big.NewInt(0).Set(result)
}
