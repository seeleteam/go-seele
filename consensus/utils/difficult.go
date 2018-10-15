/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package utils

import (
	"math/big"

	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/types"
)

// getDifficult adjust difficult by parent info
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

func VerifyDifficulty(parent *types.BlockHeader, header *types.BlockHeader) error {
	difficult := GetDifficult(header.CreateTimestamp.Uint64(), parent)
	if difficult.Cmp(header.Difficulty) != 0 {
		return consensus.ErrBlockDifficultInvalid
	}

	return nil
}
