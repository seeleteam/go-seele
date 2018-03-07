/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package seele

import (
	"bytes"
	"testing"

	"github.com/seeleteam/go-seele/common"
)

func Test_PublicSeeleAPI(t *testing.T) {
	accAddr, _ := common.GenerateRandomAddress()
	ss, _ := NewSeeleService(0, nil)
	ss.coinbase = *accAddr
	api := NewPublicSeeleAPI(ss)

	var addr common.Address
	api.Coinbase(nil, &addr)

	if !bytes.Equal(accAddr[0:], addr[0:]) {
		t.Fail()
	}
}
