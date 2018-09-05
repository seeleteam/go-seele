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

	result := map[string]interface{}{}
	err := api.GetStatus(nil, &result)

	assert.Equal(t, err, nil)
	assert.Equal(t, result["status"], "NotSyncing")
	assert.Equal(t, result["duration"], "")
	assert.Equal(t, result["startNum"], uint64(0))
	assert.Equal(t, result["amount"], uint64(0))
	assert.Equal(t, result["downloaded"], uint64(0))
}
