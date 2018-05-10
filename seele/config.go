/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
)

// Config is the seele's configuration to create seele service
type Config struct {
	TxConf core.TransactionPoolConfig

	NetworkID uint64

	Coinbase common.Address

	// genesis accounts balance info for test
	GenesisAccounts map[common.Address]*big.Int
}
