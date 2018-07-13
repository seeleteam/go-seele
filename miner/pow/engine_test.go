/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

import (
	"math/big"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/core/types"
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
}

func getDiff(interval uint64, diff *big.Int) *big.Int {
	header := &types.BlockHeader{
		CreateTimestamp: big.NewInt(0),
		Difficulty:      diff,
		Height:          10,
	}

	return GetDifficult(interval, header)
}

func Test_ValidateRewardAmount(t *testing.T) {
	var engine Engine
	amount := big.NewInt(150000000)
	var height uint64 = 0
	err := engine.ValidateRewardAmount(height, amount)
	assert.Equal(t, err, nil)

	height = 1
	err = engine.ValidateRewardAmount(height, amount)
	assert.Equal(t, err, nil)

	height = 10000
	err = engine.ValidateRewardAmount(height, amount)
	assert.Equal(t, err, nil)

	height = 62999999
	amount = big.NewInt(150000000)
	err = engine.ValidateRewardAmount(height, amount)
	assert.Equal(t, err, nil)

	height = 63000000
	amount = big.NewInt(100000000)
	err = engine.ValidateRewardAmount(height, amount)
	assert.Equal(t, err, nil)

	height = 63000000 + 10000
	amount = big.NewInt(100000000)
	err = engine.ValidateRewardAmount(height, amount)
	assert.Equal(t, err, nil)

	height = 63000000 * 2
	amount = big.NewInt(40000000)
	err = engine.ValidateRewardAmount(height, amount)
	assert.Equal(t, err, nil)

	height = 63000000 * 3
	amount = big.NewInt(40000000)
	err = engine.ValidateRewardAmount(height, amount)
	assert.Equal(t, err, nil)

	height = 63000000 * 4
	amount = big.NewInt(25000000)
	err = engine.ValidateRewardAmount(height, amount)
	assert.Equal(t, err, nil)

	height = 6300000010
	amount = big.NewInt(25000000)
	err = engine.ValidateRewardAmount(height, amount)
	assert.Equal(t, err, nil)

	height = 63000000100
	amount = big.NewInt(25000000)
	err = engine.ValidateRewardAmount(height, amount)
	assert.Equal(t, err, nil)

}
