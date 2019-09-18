package server

import (
	"crypto/ecdsa"
	"sync"

	"github.com/ethereum/go-ethereum/event"
	lru "github.com/hashicorp/golang-lru"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/bft"
	BFT "github.com/seeleteam/go-seele/consensus/bft"
	bftCore "github.com/seeleteam/go-seele/consensus/bft/core"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/log"
)

type server struct {
	config       *bft.BFTConfig
	bftEventMux  *event.TypeMux
	privateKey   *ecdsa.PrivateKey
	address      common.Address
	core         bftCore.Engine
	log          *log.SeeleLog
	db           database.Database
	chain        consensus.ChainReader
	currentBlock func() *types.Block
	hasBadBlock  func(hash common.Hash) bool

	// the channels for bft engine notifications
	commitCh          chan *types.Block
	proposedBlockHash common.Hash
	sealMu            sync.Mutex
	coreStarted       bool
	coreMu            sync.RWMutex

	// Current list of candidates we are pushing
	candidates map[common.Address]bool
	// Protects the signer fields
	candidatesLock sync.RWMutex
	// Snapshots for recent block to speed up reorgs
	recents *lru.ARCCache

	// event subscription for ChainHeadEvent event
	broadcaster consensus.Broadcaster

	recentMessages *lru.ARCCache // the cache of peer's messages
	knownMessages  *lru.ARCCache // the cache of self messages
}

// NeServer new a server for bft backend.
func NewServer(config *BFT.BFTConfig, privateKey *ecdsa.PrivateKey, db database.Database) consensus.Bft {
	recents, _ := lru.NewARC(inmemorySnapshots)
	recentMessages, _ := lru.NewARC(inmemoryPeers)
	knownMessages, _ := lru.NewARC(inmemoryMessages)
	server := &server{
		config:         config,
		bftEventMux:    new(event.TypeMux),
		privateKey:     privateKey,
		address:        cypto.PubkeyToAddress(privateKey.PublicKey),
		log:            log.GetLogger("bft"),
		db:             db,
		commitCh:       make(chan *types.Block, 1),
		recents:        recents,
		candidates:     make(map[common.Address]bool),
		coreStarted:    false,
		recentMessages: recentMessages,
		knownMessages:  knownMessages,
	}
	server.core = bftCore.NewCore(server, server.config)

}
