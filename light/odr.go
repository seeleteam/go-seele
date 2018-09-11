/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

const (
	blockRequestCode  = uint16(10)
	blockResponseCode = uint16(11)
	addTxRequestCode  = uint16(12)
	addTxResponseCode = uint16(13)
	trieRequestCode   = uint16(14)
	trieResponseCode  = uint16(15)
)

var (
	odrRequestFactories = map[uint16]func() odrRequest{
		blockRequestCode: func() odrRequest { return &odrBlock{} },
		addTxRequestCode: func() odrRequest { return &odrAddTx{} },
		trieRequestCode:  func() odrRequest { return &odrTrie{} },
	}

	odrResponseFactories = map[uint16]func() odrResponse{
		blockResponseCode: func() odrResponse { return &odrBlock{} },
		addTxResponseCode: func() odrResponse { return &odrAddTx{} },
		trieResponseCode:  func() odrResponse { return &odrTrie{} },
	}
)

type odrRequest interface {
	setRequestID(requestID uint32)                                       // set the random request ID.
	code() uint16                                                        // get request code.
	handleRequest(lp *LightProtocol) (respCode uint16, resp odrResponse) // handle the request and return response to remote peer.
	handleResponse(resp interface{})                                     // handle the received response from remote peer.
}

type odrResponse interface {
	getRequestID() uint32 // get the random request ID.
}

type odrItem struct {
	ReqID uint32
}

func (item *odrItem) getRequestID() uint32 {
	return item.ReqID
}

func (item *odrItem) setRequestID(requestID uint32) {
	item.ReqID = requestID
}
