/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

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
	blockNumberPerEra uint64 = 63000000
)

func init() {
	rewardTable := [...]float64{1.5, 1, 0.4, 0.4, 0.25}
	tailReward := float64(0.25)

	rewardTableCoin = make([]*big.Int, len(rewardTable))
	for i, r := range rewardTable {
		rewardTableCoin[i] = convertSeeleToFan(r)
	}

	tailRewardCoin = convertSeeleToFan(tailReward)
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
	} else {
		result = tailRewardCoin
	}

	return big.NewInt(0).Set(result)
}
