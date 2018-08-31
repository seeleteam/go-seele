/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

const (
	// LightProtoName protoName of Seele service
	LightProtoName = "lightSeele"

	// LightSeeleVersion version number of Seele protocol
	LightSeeleVersion uint = 1

	// BlockChainDir lightchain data directory based on config.DataRoot
	BlockChainDir = "/db/lightchain"
)

type blockQuery struct {
	ReqID  uint32      // ReqID number for request
	Hash   common.Hash // Block hash from which to retrieve (excludes Number)
	Number uint64      // Block hash from which to retrieve (excludes Hash)
}

// BlocksMsgBody represents a message struct for BlocksMsg
type BlockMsgBody struct {
	ReqID uint32
	Block *types.Block
}
