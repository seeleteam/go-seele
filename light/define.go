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

	MaxBlockHashRequest   uint64 = 1024 // maximum hases to request per message
	MaxBlockHeaderRequest uint64 = 256  // maximum headers to request per message
	MaxGapForAnnounce     uint64 = 256  // sends AnnounceQuery message if gap is more than this value
	MinHashesCached       uint64 = 256  // minimum items cached in peer for client mode
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
	Magic uint32
	Begin uint64
	End   uint64
}

type Announce struct {
	Magic           uint32
	TD              *big.Int
	CurrentBlock    common.Hash
	CurrentBlockNum uint64
	BlockNumArr     []uint64
	HeaderArr       []common.Hash
}

type HeaderHashSyncQuery struct {
	Magic    uint32
	BeginNum uint64
}

type HeaderHashSync struct {
	Magic           uint32
	TD              *big.Int
	CurrentBlock    common.Hash
	CurrentBlockNum uint64
	BeginNum        uint64
	HeaderArr       []common.Hash
}

type DownloadHeaderQuery struct {
	ReqID    uint32
	BeginNum uint64
}

type DownloadHeader struct {
	ReqID       uint32
	HasFinished bool
	Hearders    []*types.BlockHeader
}
