/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

// LocalShardNumber defines the shard number of coinbase.
// Generally, it must be initialized during program startup.
var LocalShardNumber uint

// IsShardDisabled indicates if the shard is disabled.
// THIS IS FOR UNIT TEST PURPOSE ONLY!!!
var IsShardDisabled = false
