package server

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/rpc"
)

type API struct {
	chain consensus.ChainReader
	bft   *server
}

// define all apis here
func (api *API) GetSnapshot(number *rpc.BlockNumber) (*Snapshot, error) {
	// Retrieve the requested block number (or current if none requested)
	var header *types.BlockHeader
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByHeight(uint64(number.Int64()))
	}
	// Ensure we have an actually valid block and return its snapshot
	if header == nil {
		return nil, errBlockUnknown
	}
	return api.bft.snapshot(api.chain, header.Height, header.Hash(), nil)
}

// GetSnapshotAtHash retrieves the state snapshot at a given block.
func (api *API) GetSnapshotAtHash(hash common.Hash) (*Snapshot, error) {
	header := api.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errBlockUnknown
	}
	return api.bft.snapshot(api.chain, header.Height, header.Hash(), nil)
}

// GetVerifiers retrieves the list of authorized verifiers at the specified block.
func (api *API) GetVerifiers(number *rpc.BlockNumber) ([]common.Address, error) {
	// Retrieve the requested block number (or current if none requested)
	var header *types.BlockHeader
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByHeight(uint64(number.Int64()))
	}
	// Ensure we have an actually valid block and return the verifiers from its snapshot
	if header == nil {
		return nil, errBlockUnknown
	}
	snap, err := api.bft.snapshot(api.chain, header.Height, header.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return snap.verifiers(), nil
}

// GetVerifiersAtHash retrieves the state snapshot at a given block.
func (api *API) GetVerifiersAtHash(hash common.Hash) ([]common.Address, error) {
	header := api.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errBlockUnknown
	}
	snap, err := api.bft.snapshot(api.chain, header.Height, header.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return snap.verifiers(), nil
}

// Candidates returns the current candidates the node tries to uphold and vote on.
func (api *API) Candidates() map[common.Address]bool {
	api.bft.candidatesLock.RLock()
	defer api.bft.candidatesLock.RUnlock()

	proposals := make(map[common.Address]bool)
	for address, auth := range api.bft.candidates {
		proposals[address] = auth
	}
	return proposals
}

// Propose injects a new authorization candidate that the verifier will attempt to
// push through.
func (api *API) Propose(address common.Address, auth bool) {
	api.bft.candidatesLock.Lock()
	defer api.bft.candidatesLock.Unlock()

	api.bft.candidates[address] = auth
}

// Discard drops a currently running candidate, stopping the verifier from casting
// further votes (either for or against).
func (api *API) Discard(address common.Address) {
	api.bft.candidatesLock.Lock()
	defer api.bft.candidatesLock.Unlock()

	delete(api.bft.candidates, address)
}
