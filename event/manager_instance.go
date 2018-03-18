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

// BlockMinedEventManager is event of new mined block
var BlockMinedEventManager = NewEventManager()

// TransactionInsertedEventManager is event of new transaction inserted into txpool
var TransactionInsertedEventManager = NewEventManager()

// BlockInsertedEventManager is event of new block inserted into blockchain
var BlockInsertedEventManager = NewEventManager()
