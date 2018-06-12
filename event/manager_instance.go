/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package event

// BlockDownloaderEventManager block download event
var BlockDownloaderEventManager = NewEventManager()

// block downloader event
const (
	DownloaderStartEvent  = 0
	DownloaderDoneEvent   = 1
	DownloaderFailedEvent = 2
)

// BlockMinedEventManager represents the event that a new block is mined
var BlockMinedEventManager = NewEventManager()

// TransactionInsertedEventManager represents the event that a new transaction is inserted into txpool
var TransactionInsertedEventManager = NewEventManager()

// ChainHeaderChangedEventMananger represents the event that chain header is changed
var ChainHeaderChangedEventMananger = NewEventManager()
