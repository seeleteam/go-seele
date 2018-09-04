/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
)

func Test_Dic(t *testing.T) {
	h1 := getHash("0x426225605450f355f558f89df996c0025223f08b056354d3471d64313c506dfd")
	h2 := getHash("0x6bb28d28c1a01c62ffec43dcdee6dbd2b3d4822b0301593068b62c63b49fe358")
	h3 := getHash("0x3d7c5dad1d99e0fce26ecefdd8904f304d979ff424ad666832544b72851e4b52")
	h4 := getHash("0x426225605450f355f558f89df996c0025223f08b056354d348b62c63b49fe358")

	log1 := logDist(h1, h2)
	log2 := logDist(h1, h3)
	log3 := logDist(h1, h1)
	log4 := logDist(h1, h4)

	assert.Equal(t, log1, 254)
	assert.Equal(t, log2, 255)
	assert.Equal(t, log3, 0)
	assert.Equal(t, log4, 60)
}

func getHash(s string) common.Hash {
	var h common.Hash
	buff, _ := hexutil.HexToBytes(s)
	h.SetBytes(buff)

	return h
}

func Test_Cmp(t *testing.T) {
	h1 := getHash("0x426225605450f355f558f89df996c0025223f08b056354d3471d64313c506dfd")
	h2 := getHash("0x6bb28d28c1a01c62ffec43dcdee6dbd2b3d4822b0301593068b62c63b49fe358")
	h3 := getHash("0x3d7c5dad1d99e0fce26ecefdd8904f304d979ff424ad666832544b72851e4b52")

	a := distCmp(h1, h2, h3)

	assert.Equal(t, a, -1)
	assert.Equal(t, distCmp(h1, h3, h2), 1)
	assert.Equal(t, distCmp(h1, h2, h2), 0)
}
