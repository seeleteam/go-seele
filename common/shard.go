/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

// LocalShardNumber defines the shard number of coinbase.
// Generally, it must be initialized during program startup.
var LocalShardNumber uint

// GetShardNumber calculates and returns the shard number for the specified address.
// The valid shard number is [1, ShardNumber]
func GetShardNumber(address Address) uint {
	var sum uint

	for _, b := range address.Bytes() {
		sum += uint(b)
	}

	return (sum % ShardNumber) + 1
}
