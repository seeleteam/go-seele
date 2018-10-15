/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package utils

import (
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/types"
)

func VerifyHeaderCommon(header, parent *types.BlockHeader) error {
	if header.Height != parent.Height+1 {
		return consensus.ErrBlockInvalidHeight
	}

	if header.CreateTimestamp.Cmp(parent.CreateTimestamp) < 0 {
		return consensus.ErrBlockCreateTimeOld
	}

	if err := VerifyDifficulty(parent, header); err != nil {
		return err
	}

	return nil
}
