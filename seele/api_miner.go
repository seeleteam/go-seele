/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"errors"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/miner"
)

// PrivateMinerAPI provides an API to access miner information.
type PrivateMinerAPI struct {
	s *SeeleService
}

// NewPrivateMinerAPI creates a new PrivateMinerAPI object for miner rpc service.
func NewPrivateMinerAPI(s *SeeleService) *PrivateMinerAPI {
	return &PrivateMinerAPI{s}
}

// Start API is used to start the miner with the given number of threads.
func (api *PrivateMinerAPI) Start(threads *int, result *string) error {
	if threads == nil {
		threads = new(int)
	}
	api.s.miner.SetThreads(*threads)

	if api.s.miner.IsMining() {
		return miner.ErrMinerIsRunning
	}

	return api.s.miner.Start()
}

// Status API is used to view the miner's status.
func (api *PrivateMinerAPI) Status(input *string, result *string) error {
	*result = "Unknown"
	if api.s.miner.IsMining() {
		*result = miner.ErrMinerIsRunning.Error()
	} else {
		*result = miner.ErrMinerIsStopped.Error()
	}

	return nil
}

// Stop API is used to stop the miner.
func (api *PrivateMinerAPI) Stop(input *string, result *string) error {
	if !api.s.miner.IsMining() {
		return miner.ErrMinerIsStopped
	}
	api.s.miner.Stop()

	return nil
}

// Hashrate returns the POW hashrate.
func (api *PrivateMinerAPI) Hashrate(input *string, hashrate *uint64) error {
	*hashrate = uint64(api.s.miner.Hashrate())

	return nil
}

// SetThreads  API is used to set the number of threads.
func (api *PrivateMinerAPI) SetThreads(threads *int, result *interface{}) error {
	if *threads < 0 {
		return errors.New("threads should be greater than zero.")
	}
	api.s.miner.SetThreads(*threads)

	return nil
}

// SetCoinbase API is used to set the coinbase.
func (api *PrivateMinerAPI) SetCoinbase(coinbaseStr *string, result *interface{}) error {
	coinbase, err := common.HexToAddress(*coinbaseStr)
	if err != nil {
		return err
	}
	api.s.miner.SetCoinbase(coinbase)

	return nil
}
