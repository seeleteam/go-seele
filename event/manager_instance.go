/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package event

// BlockDownloaderEvent block download event
var BlockDownloaderEventManager *EventManager = NewEventManager()

const (
	DownloaderStartEvent  = 0
	DownloaderDoneEvent   = 1
	DownloaderFailedEvent = 2
)
