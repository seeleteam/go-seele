/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/core/types"
)

var (
	// maxUint256 is a big integer representing 2^256
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

	errBlockNonceInvalid = errors.New("invalid block nonce")
)

// Engine provides the consensus operations based on POW.
type Engine struct{}

// ValidateHeader validates the specified header and returns error if validation failed.
func (engine Engine) ValidateHeader(blockHeader *types.BlockHeader) error {
	headerHash := blockHeader.Hash()
	var hashInt big.Int
	hashInt.SetBytes(headerHash.Bytes())

	target := GetMiningTarget(blockHeader.Difficulty)

	if hashInt.Cmp(target) > 0 {
		return errBlockNonceInvalid
	}

	return nil
}

// ValidateRewardAmount validates the specified amount and returns error if validation failed.
func (engine Engine) ValidateRewardAmount(blockHeight uint64, amount *big.Int) error {
	reward := GetReward(blockHeight)

	if amount == nil || amount.Cmp(reward) != 0 {
		return fmt.Errorf("invalid reward amount, block height %d, want %s, got %s", blockHeight, reward, amount)
	}

	return nil
}

// GetMiningTarget returns the mining target for the specified difficulty.
func GetMiningTarget(difficulty *big.Int) *big.Int {
	return new(big.Int).Div(maxUint256, difficulty)
}

// GetDifficult adjust difficult by parent info
func GetDifficult(time uint64, parentHeader *types.BlockHeader) *big.Int {
	// algorithm:
	// diff = parentDiff + parentDiff / 2048 * max (1 - (blockTime - parentTime) / 10, -99)
	// target block time is 10 seconds
	parentDifficult := parentHeader.Difficulty
	parentTime := parentHeader.CreateTimestamp.Uint64()
	if parentHeader.Height == 0 {
		return parentDifficult
	}

	big1 := big.NewInt(1)
	big99 := big.NewInt(-99)
	big2048 := big.NewInt(2048)

	interval := (time - parentTime) / 10
	var x *big.Int
	x = big.NewInt(int64(interval))
	x.Sub(big1, x)
	if x.Cmp(big99) < 0 {
		x = big99
	}

	var y = new(big.Int).Set(parentDifficult)
	y.Div(parentDifficult, big2048)

	var result = big.NewInt(0)
	result.Mul(x, y)
	result.Add(parentDifficult, result)

	return result
}
