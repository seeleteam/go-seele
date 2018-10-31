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
func (api *PrivatedownloaderAPI) GetStatus() *SyncInfo {
	var info SyncInfo
	api.d.getSyncInfo(&info)

	result := SyncInfo{
		Status:     info.Status,
		Duration:   info.Duration,
		StartNum:   info.StartNum,
		Amount:     info.Amount,
		Downloaded: info.Downloaded,
	}

	return &result
}

func (api *PrivatedownloaderAPI) IsSyncing() bool {
	return api.d.syncStatus != statusNone
}
