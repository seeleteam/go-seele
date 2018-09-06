/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"math/big"
	"time"

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

	forceSyncInterval = time.Second * 5 // interval time of synchronising with remote peer

	MaxBlockHashRequested uint64 = 1024
)

// statusData the structure for peers to exchange status
type statusData struct {
	ProtocolVersion uint32
	NetworkID       uint64
	IsServer        bool // whether server mode
	TD              *big.Int
	CurrentBlock    common.Hash
	CurrentBlockNum uint64
	GenesisBlock    common.Hash
}

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

type AnnounceQuery struct {
	Begin uint64
	End   uint64
}

type Announce struct {
	TD              *big.Int
	CurrentBlock    common.Hash
	CurrentBlockNum uint64
	BlockNumArr     []uint64
	HeaderArr       []common.Hash
}

type HeaderHashSyncQuery struct {
	begin uint64
}

type HeaderHashSync struct {
	TD              *big.Int
	CurrentBlock    common.Hash
	CurrentBlockNum uint64
	BeginNum        uint64
	HeaderArr       []common.Hash
}
