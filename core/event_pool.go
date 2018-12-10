package core

import (
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/listener"
	"github.com/seeleteam/go-seele/log"
)

// EventPool event pool
type EventPool struct {
	capacity int

	// this channel used to get the events from main chain
	eventsChan chan []*listener.Event

	// this version use main chain store to get receipts,
	// so use the main chain database path initialize the store.
	mainChainStore store.BlockchainStore

	log   *log.SeeleLog
	chain blockchain
	// todo add deal pools
}

// NewEventPool creates and returns an event pool.
func NewEventPool(capacity int, mainChainStore store.BlockchainStore, chain blockchain, abi *listener.ContractEventABI) *EventPool {
	log := log.GetLogger("eventpool")

	pool := &EventPool{
		capacity:       capacity,
		eventsChan:     make(chan []*listener.Event, 100),
		mainChainStore: mainChainStore,
		log:            log,
		chain:          chain,
	}

	startHeight, err := pool.getBeginHeight()
	if err != nil {
		// return no error as the chain could not deal event.
		// return event pool to avoid nil pointer.
		log.Warn("failed to get current header height, %v", err)
		return pool
	}

	// height - 1 to ensure deal the current header height
	go pool.PollingEvents(abi, startHeight-1)

	return pool
}

func (pool *EventPool) getBeginHeight() (uint64, error) {
	store := pool.mainChainStore
	hash, err := store.GetHeadBlockHash()
	if err != nil {
		return 0, errors.NewStackedError(err, "failed to get block hash")
	}

	header, err := store.GetBlockHeader(hash)
	if err != nil {
		return 0, errors.NewStackedError(err, "failed to get block header")
	}

	return header.Height, nil
}

// PollingEvents is used to poll for events from main chain.
func (pool *EventPool) PollingEvents(abi *listener.ContractEventABI, beginHeight uint64) {
	if abi == nil {
		pool.log.Debug("no contract event to listen")
		return
	}

	var (
		store        = pool.mainChainStore
		targetHeight = beginHeight
		headerHeight uint64
		err          error
	)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// get current header height
			headerHeight, err = pool.getBeginHeight()
			if err != nil {
				pool.log.Error("failed to get current header height, %v", err)
				continue
			}

			// avoid duplicate blocks request
			if targetHeight >= headerHeight {
				continue
			}

			// avoid skip block
			targetHeight++

			// get the confirmed blocks over ConfirmedBlockNumber
			if targetHeight <= common.ConfirmedBlockNumber {
				continue
			}

			blockHash, err := store.GetBlockHash(targetHeight - common.ConfirmedBlockNumber)
			if err != nil {
				pool.log.Error("failed to get confirmed block hash, %v", err)
				continue
			}

			receipts, err := store.GetReceiptsByBlockHash(blockHash)
			if err != nil {
				pool.log.Error("failed to get receipts by block hash, %v", err)
				continue
			}

			events, err := abi.GetEvents(receipts)
			if err != nil {
				pool.log.Error("failed to get events from receipts, %v", err)
				continue
			}

			if events == nil {
				continue
			}

			pool.eventsChan <- events
		}
	}
}
