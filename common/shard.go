/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

// LocalShardNumber defines the shard number of coinbase.
// Generally, it must be initialized during program startup.
var LocalShardNumber uint

// IsShardDisabled indicates if the shard is disabled.
// THIS IS FOR TEST PURPOSE ONLY!!!
var IsShardDisabled = false

// GetShardNumber calculates and returns the shard number for the specified address.
// The valid shard number is [1, ShardNumber], or 0 if IsShardDisabled is true.
func GetShardNumber(address Address) uint {
	if IsShardDisabled {
		return 0
	}

	var sum uint

	for _, b := range address.Bytes() {
		sum += uint(b)
	}

	return (sum % ShardNumber) + 1
}
