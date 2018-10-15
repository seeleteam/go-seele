/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package utils

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/core/types"
	"github.com/stretchr/testify/assert"
)

func Test_GetDifficult(t *testing.T) {
	diff1 := big.NewInt(10000)
	diff2 := getDiff(5, diff1)
	assert.Equal(t, diff2.Int64(), int64(10004))

	diff3 := getDiff(16, diff1)
	assert.Equal(t, diff3.Int64(), int64(10000))

	diff4 := getDiff(24, diff1)
	assert.Equal(t, diff4.Int64(), int64(9996))

	diff5 := getDiff(11000, diff1)
	assert.Equal(t, diff5.Int64(), int64(9604))

	diff6 := getDiff(100, diff1)
	assert.Equal(t, diff6.Int64(), int64(9964))

	diff7 := getDiffWithHeight(100, diff1, 0)
	assert.Equal(t, diff7.Int64(), diff1.Int64())
}

func getDiff(interval uint64, diff *big.Int) *big.Int {
	return getDiffWithHeight(interval, diff, 10)
}

func getDiffWithHeight(interval uint64, diff *big.Int, height uint64) *big.Int {
	header := &types.BlockHeader{
		CreateTimestamp: big.NewInt(0),
		Difficulty:      diff,
		Height:          height,
	}

	return GetDifficult(interval, header)
}
