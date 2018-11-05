/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

import (
	"testing"

	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func Test_Download_NewPrivatedownloaderAPI(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)

	api := NewPrivatedownloaderAPI(dl)
	assert.Equal(t, api != nil, true)
	assert.Equal(t, api.d, dl)
}

func Test_Download_GetStatus(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)
	api := NewPrivatedownloaderAPI(dl)

	result := api.GetStatus()

	assert.Equal(t, result.Status, "NotSyncing")
	assert.Equal(t, result.Duration, "")
	assert.Equal(t, result.StartNum, uint64(0))
	assert.Equal(t, result.Amount, uint64(0))
	assert.Equal(t, result.Downloaded, uint64(0))
}
