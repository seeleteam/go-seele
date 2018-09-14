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

func (bc *LightChain) GetState(root common.Hash) (*state.Statedb, error) {
	trie := newOdrTrie(bc.odrBackend, root, state.TrieDbPrefix)
	return state.NewStatedbWithTrie(trie), nil
}

// CurrentHeader returns the HEAD block header of the blockchain.
func (bc *LightChain) CurrentHeader() *types.BlockHeader {
	hash, err := bc.bcStore.GetHeadBlockHash()
	if err != nil {
		return nil
	}

	header, err := bc.bcStore.GetBlockHeader(hash)
	if err != nil {
		return nil
	}
	return header
}

func (bc *LightChain) GetStore() store.BlockchainStore {
	return bc.bcStore
}

// WriteHeader writes the specified block header to the blockchain.
func (bc *LightChain) WriteHeader(header *types.BlockHeader) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if err := core.ValidateBlockHeader(header, bc.engine, bc.bcStore); err != nil {
		return err
	}

	previousTd, err := bc.bcStore.GetBlockTotalDifficulty(header.PreviousBlockHash)
	if err != nil {
		return err
	}

	currentTd := new(big.Int).Add(previousTd, header.Difficulty)
	isHead := currentTd.Cmp(bc.canonicalTD) > 0

	if err := bc.bcStore.PutBlockHeader(header.Hash(), header, currentTd, isHead); err != nil {
		return err
	}

	if isHead {
		if err := core.DeleteLargerHeightBlocks(bc.bcStore, header.Height+1, nil); err != nil {
			return err
		}

		if err := core.OverwriteStaleBlocks(bc.bcStore, header.PreviousBlockHash, nil); err != nil {
			return err
		}

		bc.canonicalTD = currentTd
		bc.currentHeader = header
	}

	return nil
}

func (bc LightChain) GetCurrentState() (*state.Statedb, error) {
	return bc.GetState(bc.currentHeader.StateHash)
}
