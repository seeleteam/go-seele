/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsShardEnabled(t *testing.T) {
	// To store the "LocalShardNumber" so as not to impact other test cases that depend on this global variable.
	orginalLocalShardNumber := LocalShardNumber

	defer func() {
		// Restore
		LocalShardNumber = orginalLocalShardNumber
	}()

	// IsShardEnabled returns true if the LocalShardNumber is set and less than ShardCount+1.
	// Otherwise, false.
	LocalShardNumber = 0
	assert.Equal(t, IsShardEnabled(), false)

	LocalShardNumber = ShardCount + 1
	assert.Equal(t, IsShardEnabled(), false)

	for shard := uint(1); shard <= ShardCount; shard++ {
		LocalShardNumber = shard
		assert.Equal(t, IsShardEnabled(), true)
	}
}
