/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import "errors"

const (
	blockRequestCode uint16 = 10 + iota
	blockResponseCode
	addTxRequestCode
	addTxResponseCode
	trieRequestCode
	trieResponseCode
	receiptRequestCode
	receiptResponseCode
	txByHashRequestCode
	txByHashResponseCode
	protocolMsgCodeLength // protocolMsgCodeLength always defined in the end.
)

var (
	odrRequestFactories = map[uint16]func() odrRequest{
		blockRequestCode:    func() odrRequest { return &odrBlock{} },
		addTxRequestCode:    func() odrRequest { return &odrAddTx{} },
		trieRequestCode:     func() odrRequest { return &odrTriePoof{} },
		receiptRequestCode:  func() odrRequest { return &odrtReceipt{} },
		txByHashRequestCode: func() odrRequest { return &odrTxByHash{} },
	}

	odrResponseFactories = map[uint16]func() odrResponse{
		blockResponseCode:    func() odrResponse { return &odrBlock{} },
		addTxResponseCode:    func() odrResponse { return &odrAddTx{} },
		trieResponseCode:     func() odrResponse { return &odrTriePoof{} },
		receiptResponseCode:  func() odrResponse { return &odrtReceipt{} },
		txByHashResponseCode: func() odrResponse { return &odrTxByHash{} },
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
	getError() error      // get the response error if any.
}

type odrItem struct {
	ReqID uint32 // random request ID that generated dynamically
	Error string // response error
}

func (item *odrItem) getRequestID() uint32 {
	return item.ReqID
}

func (item *odrItem) setRequestID(requestID uint32) {
	item.ReqID = requestID
}

func (item *odrItem) getError() error {
	if len(item.Error) == 0 {
		return nil
	}

	return errors.New(item.Error)
}
