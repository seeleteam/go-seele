/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

// PublicdownloaderAPI provides an API to access downloader information.
type PublicdownloaderAPI struct {
	d *Downloader
}

// NewPublicdownloaderAPI creates a new PublicdownloaderAPI object for rpc service.
func NewPublicdownloaderAPI(d *Downloader) *PublicdownloaderAPI {
	return &PublicdownloaderAPI{d}
}

// SyncInfo sync information for current downloader sessoin.
type SyncInfo struct {
	Status     string // readable string of downloader.syncStatus
	Duration   string // duration in seconds
	StartNum   uint64 // start block number
	Amount     uint64 // amount of blocks need to download
	Downloaded uint64
}

// GetStatus gets the SyncInfo.
func (api *PublicdownloaderAPI) GetStatus(input interface{}, info *SyncInfo) error {
	api.d.getSyncInfo(info)
	return nil
}
