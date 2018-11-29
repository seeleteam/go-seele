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

// IsLightMode reprsents whethre node is light node or full node.
var IsLightMode bool

// IsShardEnabled returns true if the LocalShardNumber is set. Otherwise, false.
func IsShardEnabled() bool {
	return LocalShardNumber > UndefinedShardNumber && LocalShardNumber <= ShardCount
}
