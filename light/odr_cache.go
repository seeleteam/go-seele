/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"reflect"

	"github.com/hashicorp/golang-lru"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

const (
	cacheSizeOdrTx = 1024
)

type odrCache struct {
	typedCaches map[string]*lru.Cache
}

func newOdrCache() *odrCache {
	return &odrCache{
		typedCaches: map[string]*lru.Cache{
			odrRequestType(&odrTxByHashRequest{}): common.MustNewCache(cacheSizeOdrTx),
		},
	}
}

func odrRequestType(request odrRequest) string {
	return reflect.TypeOf(request).String()
}

func odrRequestKey(request odrRequest) common.Hash {
	reqID := request.getRequestID()
	defer request.setRequestID(reqID)

	request.setRequestID(0)
	return crypto.MustHash([]interface{}{odrRequestType(request), request})
}

func (cache *odrCache) get(request odrRequest) (odrResponse, bool) {
	reqType := odrRequestType(request)
	lruCache := cache.typedCaches[reqType]
	if lruCache == nil {
		return nil, false
	}

	reqKey := odrRequestKey(request)
	if resp, ok := lruCache.Get(reqKey); ok {
		return resp.(odrResponse), true
	}

	return nil, false
}

func (cache *odrCache) add(request odrRequest, response odrResponse) {
	reqType := odrRequestType(request)
	if lruCache := cache.typedCaches[reqType]; lruCache != nil {
		reqKey := odrRequestKey(request)
		lruCache.Add(reqKey, response)
	}
}
