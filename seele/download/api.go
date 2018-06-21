/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

// PrivatedownloaderAPI provides an API to access downloader information.
type PrivatedownloaderAPI struct {
	d *Downloader
}

// NewPrivatedownloaderAPI creates a new PrivatedownloaderAPI object for rpc service.
func NewPrivatedownloaderAPI(d *Downloader) *PrivatedownloaderAPI {
	return &PrivatedownloaderAPI{d}
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
func (api *PrivatedownloaderAPI) GetStatus(input interface{}, result *map[string]interface{}) error {
	var info SyncInfo
	api.d.getSyncInfo(&info)

	*result = map[string]interface{}{
		"status":     info.Status,
		"duration":   info.Duration,
		"startNum":   info.StartNum,
		"amount":     info.Amount,
		"downloaded": info.Downloaded,
	}

	return nil
}
