/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
)

// Config is the seele's configuration to create seele service
type Config struct {
	TxConf core.TransactionPoolConfig

	NetworkID uint64

	// DataRoot root dir of local storage
	DataRoot string
	Coinbase common.Address `toml:",omitempty"`
}
