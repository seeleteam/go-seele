/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"math/big"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner/pow"
)

// LightChain represents a canonical chain that by default only handles block headers.
type LightChain struct {
	mutex         sync.RWMutex
	bcStore       store.BlockchainStore
	odrBackend    *odrBackend
	engine        core.ConsensusEngine
	currentHeader *types.BlockHeader
	canonicalTD   *big.Int
	log           *log.SeeleLog
}

func newLightChain(bcStore store.BlockchainStore, lightDB database.Database, odrBackend *odrBackend) (*LightChain, error) {
	chain := &LightChain{
		bcStore:    bcStore,
		odrBackend: odrBackend,
		engine:     &pow.Engine{},
		log:        log.GetLogger("LightChain"),
	}

	currentHeaderHash, err := bcStore.GetHeadBlockHash()
	if err != nil {
		return nil, err
	}

	chain.currentHeader, err = bcStore.GetBlockHeader(currentHeaderHash)
	if err != nil {
		return nil, err
	}

	td, err := bcStore.GetBlockTotalDifficulty(currentHeaderHash)
	if err != nil {
		return nil, err
	}

	chain.canonicalTD = td

	return chain, nil
}

// CurrentBlock @todo
func (lc *LightChain) CurrentBlock() *types.Block {
	return nil
}

// GetState get statedb
func (lc *LightChain) GetState(root common.Hash) (*state.Statedb, error) {
	trie := newOdrTrie(lc.odrBackend, root, state.TrieDbPrefix)

	return state.NewStatedbWithTrie(trie), nil
}

// GetStore get underlying store
func (lc *LightChain) GetStore() store.BlockchainStore {
	return lc.bcStore
}

// WriteHeader writes the specified block header to the blockchain.
func (lc *LightChain) WriteHeader(header *types.BlockHeader) error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	if err := core.ValidateBlockHeader(header, lc.engine, lc.bcStore); err != nil {
		return err
	}

	previousTd, err := lc.bcStore.GetBlockTotalDifficulty(header.PreviousBlockHash)
	if err != nil {
		return err
	}

	currentTd := new(big.Int).Add(previousTd, header.Difficulty)
	isHead := currentTd.Cmp(lc.canonicalTD) > 0

	if err := lc.bcStore.PutBlockHeader(header.Hash(), header, currentTd, isHead); err != nil {
		return err
	}

	if isHead {
		if err := core.DeleteLargerHeightBlocks(lc.bcStore, header.Height+1, nil); err != nil {
			return err
		}

		if err := core.OverwriteStaleBlocks(lc.bcStore, header.PreviousBlockHash, nil); err != nil {
			return err
		}

		lc.canonicalTD = currentTd
		lc.currentHeader = header
	}

	return nil
}

// GetCurrentState get current state
func (lc *LightChain) GetCurrentState() (*state.Statedb, error) {
	return lc.GetState(lc.currentHeader.StateHash)
}
