/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

var (
	// rewardTable the reward value is per year. Which means the first value is for first year, second value is for second year, etc...
	rewardTable = [...]int64{200, 100, 50, 40, 30}

	// tailReward it is used when out of the reward table. we use a constant reward value.
	tailReward int64 = 30

	// blockNumberGeneratePerYear block number generated per year.
	blockNumberGeneratePerYear uint64 = 525000
)

// GetReward get reward amount according to block height
func GetReward(blockHeight uint64) int64 {
	era := int(blockHeight / blockNumberGeneratePerYear)

	if era < len(rewardTable) {
		return rewardTable[era]
	}

	return tailReward
}
