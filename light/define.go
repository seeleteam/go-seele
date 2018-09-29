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

	// MaxBlockHashRequest maximum hashes to request per message
	MaxBlockHashRequest uint64 = 1024

	// MaxBlockHeaderRequest maximum headers to request per message
	MaxBlockHeaderRequest uint64 = 256

	// MaxGapForAnnounce sends AnnounceQuery message if gap is more than this value
	MaxGapForAnnounce uint64 = 256

	// MinHashesCached minimum items cached in peer for client mode
	MinHashesCached uint64 = 256

	forceSyncInterval = time.Second * 5 // interval time of synchronising with remote peer
)

// statusData the structure for peers to exchange status
type statusData struct {
	ProtocolVersion uint32
	NetworkID       string
	IsServer        bool // whether server mode
	TD              *big.Int
	CurrentBlock    common.Hash
	CurrentBlockNum uint64
	GenesisBlock    common.Hash
}

// AnnounceQuery header of AnnounceQuery request
type AnnounceQuery struct {
	Magic uint32
	Begin uint64
	End   uint64
}

// AnnounceBody body of AnnounceQuery response
type AnnounceBody struct {
	Magic           uint32
	TD              *big.Int
	CurrentBlock    common.Hash
	CurrentBlockNum uint64
	BlockNumArr     []uint64
	HeaderArr       []common.Hash
}

// HeaderHashSyncQuery header of HeaderHashSyncQuery request
type HeaderHashSyncQuery struct {
	Magic    uint32
	BeginNum uint64
}

// HeaderHashSync body of HeaderHashSyncQuery response
type HeaderHashSync struct {
	Magic           uint32
	TD              *big.Int
	CurrentBlock    common.Hash
	CurrentBlockNum uint64
	BeginNum        uint64
	HeaderArr       []common.Hash
}

// DownloadHeaderQuery header of DownloadHeaderQuery request
type DownloadHeaderQuery struct {
	ReqID    uint32
	BeginNum uint64
}

// DownloadHeader body of DownloadHeaderQuery response
type DownloadHeader struct {
	ReqID       uint32
	HasFinished bool
	Hearders    []*types.BlockHeader
}
