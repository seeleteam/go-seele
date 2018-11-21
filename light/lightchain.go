/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"math/big"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
)

// LightChain represents a canonical chain that by default only handles block headers.
type LightChain struct {
	mutex                     sync.RWMutex
	bcStore                   store.BlockchainStore
	odrBackend                *odrBackend
	engine                    consensus.Engine
	currentHeader             *types.BlockHeader
	canonicalTD               *big.Int
	headerChangedEventManager *event.EventManager
	log                       *log.SeeleLog
}

func newLightChain(bcStore store.BlockchainStore, lightDB database.Database, odrBackend *odrBackend, engine consensus.Engine) (*LightChain, error) {
	chain := &LightChain{
		bcStore:    bcStore,
		odrBackend: odrBackend,
		engine:     engine,
		headerChangedEventManager: event.NewEventManager(),
		log: log.GetLogger("LightChain"),
	}

	currentHeaderHash, err := bcStore.GetHeadBlockHash()
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to get HEAD block hash")
	}

	chain.currentHeader, err = bcStore.GetBlockHeader(currentHeaderHash)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get block header, hash = %v", currentHeaderHash)
	}

	td, err := bcStore.GetBlockTotalDifficulty(currentHeaderHash)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get block TD, hash = %v", currentHeaderHash)
	}

	chain.canonicalTD = td

	return chain, nil
}

// GetState get statedb
func (lc *LightChain) GetState(root common.Hash) (*state.Statedb, error) {
	panic("unsupported")
}

func (lc *LightChain) GetStateByRootAndBlockHash(root, blockHash common.Hash) (*state.Statedb, error) {
	trie := newOdrTrie(lc.odrBackend, root, state.TrieDbPrefix, blockHash)
	return state.NewStatedbWithTrie(trie), nil
}

// CurrentHeader returns the HEAD block header of the blockchain.
func (lc *LightChain) CurrentHeader() *types.BlockHeader {
	return lc.currentHeader
}

// GetStore get underlying store
func (lc *LightChain) GetStore() store.BlockchainStore {
	return lc.bcStore
}

// GetHeader retrieves a block header from the database by hash and number.
func (lc *LightChain) GetHeaderByHeight(height uint64) *types.BlockHeader {
	hash, err := lc.bcStore.GetBlockHash(height)
	if err != nil {
		lc.log.Warn("get block header by height failed, err %s. height %d", err, height)
		return nil
	}

	return lc.GetHeaderByHash(hash)
}

// GetHeaderByNumber retrieves a block header from the database by number.
func (lc *LightChain) GetHeaderByHash(hash common.Hash) *types.BlockHeader {
	header, err := lc.bcStore.GetBlockHeader(hash)
	if err != nil {
		lc.log.Warn("get block header by hash failed, err %s", err)
		return nil
	}

	return header
}

// GetHeaderByHash
func (lc *LightChain) GetBlockByHash(hash common.Hash) *types.Block {
	// this is only provide for miner interface. for light chain, there is no mining, so just return nil.
	return nil
}

// WriteHeader writes the specified block header to the blockchain.
func (lc *LightChain) WriteHeader(header *types.BlockHeader) error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	if err := core.ValidateBlockHeader(header, lc.engine, lc.bcStore, lc); err != nil {
		return errors.NewStackedError(err, "failed to validate block header")
	}

	previousTd, err := lc.bcStore.GetBlockTotalDifficulty(header.PreviousBlockHash)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get block TD, hash = %v", header.PreviousBlockHash)
	}

	currentTd := new(big.Int).Add(previousTd, header.Difficulty)
	isHead := currentTd.Cmp(lc.canonicalTD) > 0

	if err := lc.bcStore.PutBlockHeader(header.Hash(), header, currentTd, isHead); err != nil {
		return errors.NewStackedErrorf(err, "failed to put block header, header = %+v", header)
	}

	if !isHead {
		return nil
	}

	if err := core.DeleteLargerHeightBlocks(lc.bcStore, header.Height+1, nil); err != nil {
		return errors.NewStackedErrorf(err, "failed to delete larger height blocks in canonical chain, height = %v", header.Height+1)
	}

	if err := core.OverwriteStaleBlocks(lc.bcStore, header.PreviousBlockHash, nil); err != nil {
		return errors.NewStackedErrorf(err, "failed to overwrite stale blocks in old canonical chain, hash = %v", header.PreviousBlockHash)
	}

	lc.canonicalTD = currentTd
	lc.currentHeader = header

	lc.headerChangedEventManager.Fire(header)

	return nil
}

// GetCurrentState get current state
func (lc *LightChain) GetCurrentState() (*state.Statedb, error) {
	return lc.GetStateByRootAndBlockHash(lc.currentHeader.StateHash, lc.currentHeader.Hash())
}
