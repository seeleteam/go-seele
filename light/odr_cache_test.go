/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/stretchr/testify/assert"
)

func Test_odrCache_odrRequestType(t *testing.T) {
	assert.Equal(t, "*light.odrTxByHashRequest", odrRequestType(&odrTxByHashRequest{}))
}

func Test_odrCache_odrRequestKey(t *testing.T) {
	req := &odrTxByHashRequest{
		OdrItem: OdrItem{
			ReqID: 38,
		},
		TxHash: common.StringToHash("tx hash"),
	}

	hash1 := odrRequestKey(req)

	// odr request should not be changed.
	assert.Equal(t, uint32(38), req.ReqID)
	assert.Equal(t, common.StringToHash("tx hash"), req.TxHash)

	req2 := &odrTxByHashRequest{
		OdrItem: OdrItem{
			ReqID: 46,
		},
		TxHash: common.StringToHash("tx hash"),
	}
	hash2 := odrRequestKey(req2)

	// 2 requests with same txHash should have the same key,
	// even the request ID is different.
	assert.Equal(t, hash1, hash2)
}

func Test_odrCache_addGet(t *testing.T) {
	cache := newOdrCache()
	var requests []odrRequest
	var responses []odrResponse

	// add request/response, so that LRU cache is full.
	for i := 0; i < cacheSizeOdrTx; i++ {
		req := &odrTxByHashRequest{
			OdrItem: OdrItem{
				ReqID: 38,
			},
			TxHash: common.BigToHash(big.NewInt(int64(i))),
		}
		requests = append(requests, req)

		resp := &odrTxByHashResponse{
			OdrItem: req.OdrItem,
		}
		responses = append(responses, resp)

		cache.add(req, resp)
	}

	assert.Equal(t, cacheSizeOdrTx, len(requests))
	assert.Equal(t, cacheSizeOdrTx, len(responses))

	// succeed to get response from cache even request ID changed.
	for i := 0; i < cacheSizeOdrTx; i++ {
		requests[i].setRequestID(777)
		resp, ok := cache.get(requests[i])
		assert.True(t, ok)
		assert.Equal(t, responses[i], resp)
	}

	// exceed the LRU cache capacity.
	req := &odrTxByHashRequest{
		OdrItem: OdrItem{
			ReqID: 38,
		},
		TxHash: common.BigToHash(big.NewInt(cacheSizeOdrTx)),
	}
	requests = append(requests, req)

	resp := &odrTxByHashResponse{
		OdrItem: req.OdrItem,
	}
	responses = append(responses, resp)

	cache.add(req, resp)

	// failed to get response from cache for the request with txhash 0.
	req.TxHash = common.BigToHash(big.NewInt(0))
	resp2, ok := cache.get(req)
	assert.False(t, ok)
	assert.Nil(t, resp2)

	// succeed to get response from cache for the requests with txhash [1,cacheSizeOdrTx]
	for i := 1; i <= cacheSizeOdrTx; i++ {
		req.TxHash = common.BigToHash(big.NewInt(int64(i)))
		resp2, ok = cache.get(req)
		assert.True(t, ok)
		assert.Equal(t, responses[i], resp2)
	}
}
