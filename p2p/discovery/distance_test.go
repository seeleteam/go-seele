/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/log"
	"testing"
)

func Test_Dic(t *testing.T) {
	h1 := getHash("0x426225605450f355f558f89df996c0025223f08b056354d3471d64313c506dfd")
	h2 := getHash("0x6bb28d28c1a01c62ffec43dcdee6dbd2b3d4822b0301593068b62c63b49fe358")
	h3 := getHash("0x3d7c5dad1d99e0fce26ecefdd8904f304d979ff424ad666832544b72851e4b52")
	h4 := getHash("0x426225605450f355f558f89df996c0025223f08b056354d348b62c63b49fe358")

	log1 := logdist(h1, h2)
	log2 := logdist(h1, h3)
	log3 := logdist(h1, h1)
	log4 := logdist(h1, h4)

	assert.Equal(t, log3, 0)

	log.Debug("%d, %d, %d", log1, log2, log4)
}

func getHash(s string) *common.Hash  {
	var h common.Hash
	buff, _ := hexutil.HexToBytes(s)
	h.SetBytes(buff)

	return &h
}

func Test_Cmp(t *testing.T) {
	h1 := getHash("0x426225605450f355f558f89df996c0025223f08b056354d3471d64313c506dfd")
	h2 := getHash("0x6bb28d28c1a01c62ffec43dcdee6dbd2b3d4822b0301593068b62c63b49fe358")
	h3 := getHash("0x3d7c5dad1d99e0fce26ecefdd8904f304d979ff424ad666832544b72851e4b52")

	a := distcmp(h1, h2, h3)
	log.Debug("%d", a)
	assert.Equal(t, a, -1)

	assert.Equal(t, distcmp(h1, h3, h2), 1)

	assert.Equal(t, distcmp(h1, h2, h2), 0)
}