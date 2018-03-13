/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package event

// BlockDownloaderEvent block download event
var BlockDownloaderEventManager = NewEventManager()

// block downloader event
const (
	DownloaderStartEvent  = 0
	DownloaderDoneEvent   = 1
	DownloaderFailedEvent = 2
)

// event of new mined block
var BlockMinedEventManager = NewEventManager()

// event of new transaction inserted into txpool
var TransactionInsertedEventManager = NewEventManager()

// event of new block inserted into blockchain
var BlockInsertedEventManager = NewEventManager()