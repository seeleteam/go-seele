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
	accAddr := common.HexToAddress("0x0548d0b1a3297fea072284f86b9fd39a9f1273c46fba8951b62de5b95cd3dd846278057ec4df598a0b089a0bdc0c8fd3aa601cf01a9f30a60292ea0769388d1f")
	ss, _ := NewSeeleService(0, nil)
	ss.coinbase = accAddr
	api := NewPublicSeeleAPI(ss)

	var addr common.Address
	api.Coinbase(nil, &addr)

	if !bytes.Equal(accAddr[0:], addr[0:]) {
		t.Fail()
	}
}
