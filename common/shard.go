/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

// UndefinedShardNumber is the default value if shard number not specified.
const UndefinedShardNumber = uint(0)

// LocalShardNumber defines the shard number of coinbase.
// Generally, it must be initialized during program startup.
var LocalShardNumber uint

// IsShardEnabled returns true if the LocalShardNumber is set. Otherwise, false.
func IsShardEnabled() bool {
	return LocalShardNumber > UndefinedShardNumber && LocalShardNumber <= ShardCount
}
