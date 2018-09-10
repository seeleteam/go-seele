/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

var (
	odrRequestFactories = map[uint16]func() odrRequest{}

	odrResponseFactories = map[uint16]func() odrResponse{}
)

type odrRequest interface {
	setRequestID(requestID uint32)             // random request ID.
	code() uint16                              // request code.
	handleRequest() (odrResponse, error)       // handle the request and return response to remote peer.
	handleResponse(response interface{}) error // handle the received response from remote peer.
}

type odrResponse interface {
	getRequestID() uint32 // random request ID.
	code() uint16         // response code.
}
