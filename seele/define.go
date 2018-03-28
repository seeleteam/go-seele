/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
)

const (
	// SeeleProtoName protoName of Seele service
	SeeleProtoName = "seele"

	// SeeleVersion Version number of Seele protocol
	SeeleVersion uint = 1

	// BlockChainDir blockchain data directory based on config.DataRoot
	BlockChainDir = "/db/blockchain"
)

const (
	StatusMsg = 0x00
)

// statusData the structure for peers to exchange status
type statusData struct {
	ProtocolVersion uint32
	NetworkID       uint64
	TD              *big.Int
	CurrentBlock    common.Hash
	GenesisBlock    common.Hash
}

// getBlockHeadersData represents a block header query.
type getBlockHeadersData struct {
	Hash    common.Hash // Block hash from which to retrieve headers (excludes Number)
	Number  uint64      // Block hash from which to retrieve headers (excludes Hash)
	Amount  uint64      // Maximum number of headers to retrieve
	Reverse bool        // Query direction (false = rising towards latest, true = falling towards genesis)
}
