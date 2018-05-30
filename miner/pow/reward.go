/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

import (
	"github.com/seeleteam/go-seele/common"
	"math/big"
)

var (
	// rewardTable the reward value is per year. Which means the first value is for first year, second value is for second year, etc...
	rewardTableCoin []*big.Int

	// tailReward it is used when out of the reward table. we use a constant reward value.
	tailRewardCoin *big.Int

	// blockNumberPerEra block number per reward era. It is approximation of block number generated per year.
	blockNumberPerEra uint64 = 525000 * 4
	//SeeleToCoin base coin number
	SeeleToCoin = common.SeeleToCoin
)

func init() {
	rewardTable := [...]int64{200, 100, 50, 40, 30}
	tailReward := int64(30)

	rewardTableCoin = make([]*big.Int, len(rewardTable))
	for i, r := range rewardTable {
		seele := big.NewInt(r)
		rewardTableCoin[i] = big.NewInt(0).Mul(seele, SeeleToCoin)
	}

	reward := big.NewInt(tailReward)
	tailRewardCoin = big.NewInt(0).Mul(reward, SeeleToCoin)
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
