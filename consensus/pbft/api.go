/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

type API struct {
	engine *PBFTEngine
}

// GetHashrate returns the current hashrate for local CPU miner and remote miner.
func (api *API) GetHashrate() uint64 {
	return uint64(api.engine.hashrate.Rate1())
}

// GetThreads returns the thread number of the miner engine
func (api *API) GetThreads() int {
	return api.engine.threads
}
