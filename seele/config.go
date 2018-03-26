/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
)

type Config struct {
	txConf core.TransactionPoolConfig

	NetworkID uint64

	// DataRoot root dir of local storage
	DataRoot string
	Coinbase common.Address `toml:",omitempty"`
}
