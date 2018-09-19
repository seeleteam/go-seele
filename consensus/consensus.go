/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package consensus

import (
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
)

type Engine interface {
	// Prepare header before generate block
	Prepare(store store.BlockchainStore, header *types.BlockHeader) error

	// VerifyHeader verify block header
	VerifyHeader(store store.BlockchainStore, header *types.BlockHeader) error

	// Seal generate block
	Seal(store store.BlockchainStore, block *types.Block, stop <-chan struct{}, results chan<- *types.Block) error

	// GetEngineInfo get engine basic info
	GetEngineInfo() map[string]interface{}
}
