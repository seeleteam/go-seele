/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

import (
	"errors"
	"math/big"

	"github.com/seeleteam/go-seele/core/types"
)

// MinerRewardAmount specifies the amount rewarded when the miner generates a new block
const MinerRewardAmount = 10

var (
	// maxUint256 is a big integer representing 2^256
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

	constMinerRewardAmount = big.NewInt(MinerRewardAmount)

	errRewardAmountInvalid = errors.New("invalid reward amount")
	errBlockNonceInvalid   = errors.New("invalid block nonce")
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
func (engine Engine) ValidateRewardAmount(amount *big.Int) error {
	if amount == nil || amount.Cmp(constMinerRewardAmount) != 0 {
		return errRewardAmountInvalid
	}

	return nil
}

// GetMiningTarget returns the mining target for the specified difficulty.
func GetMiningTarget(difficulty *big.Int) *big.Int {
	return new(big.Int).Div(maxUint256, difficulty)
}
