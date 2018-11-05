/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"errors"

	"github.com/seeleteam/go-seele/core/store"
)

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
	debtRequestCode
	debtResponseCode
	protocolMsgCodeLength // protocolMsgCodeLength always defined in the end.
)

var (
	odrRequestFactories = map[uint16]func() odrRequest{
		blockRequestCode:    func() odrRequest { return &odrBlock{} },
		addTxRequestCode:    func() odrRequest { return &odrAddTx{} },
		trieRequestCode:     func() odrRequest { return &odrTriePoof{} },
		receiptRequestCode:  func() odrRequest { return &odrReceiptRequest{} },
		txByHashRequestCode: func() odrRequest { return &odrTxByHashRequest{} },
		debtRequestCode:     func() odrRequest { return &odrDebtRequest{} },
	}

	odrResponseFactories = map[uint16]func() odrResponse{
		blockResponseCode:    func() odrResponse { return &odrBlock{} },
		addTxResponseCode:    func() odrResponse { return &odrAddTx{} },
		trieResponseCode:     func() odrResponse { return &odrTriePoof{} },
		receiptResponseCode:  func() odrResponse { return &odrReceiptResponse{} },
		txByHashResponseCode: func() odrResponse { return &odrTxByHashResponse{} },
		debtResponseCode:     func() odrResponse { return &odrDebtResponse{} },
	}
)

type odrRequest interface {
	getRequestID() uint32                                         // get the random request ID.
	setRequestID(requestID uint32)                                // set the random request ID.
	code() uint16                                                 // get request code.
	handle(lp *LightProtocol) (respCode uint16, resp odrResponse) // handle the request and return response to remote peer.
}

type odrResponse interface {
	getRequestID() uint32                                             // get the random request ID.
	getError() error                                                  // get the response error if any.
	validate(request odrRequest, bcStore store.BlockchainStore) error // validate the retrieved response.
}

// OdrItem is base struct for ODR request and response.
type OdrItem struct {
	ReqID uint32 // random request ID that generated dynamically
	Error string // response error
}

func (item *OdrItem) getRequestID() uint32 {
	return item.ReqID
}

func (item *OdrItem) setRequestID(requestID uint32) {
	item.ReqID = requestID
}

func (item *OdrItem) getError() error {
	if len(item.Error) == 0 {
		return nil
	}

	return errors.New(item.Error)
}
