/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

const (
	// SeeleProtoName protoName of Seele service
	SeeleProtoName = "seele"

	// SeeleVersion Version number of Seele protocol
	SeeleVersion uint = 1

	// DownloadStartEvent event name of start downloading blocks
	DownloadStartEvent = "downloader.StartEvent"

	// DownloadDownEvent event name of blocks downloaded successfully
	DownloadDownEvent = "downloader.DoneEvent"

	// DownloadFailedEvent event name of blocks downloaded failed
	DownloadFailedEvent = "downloader.FailedEvent"
)
