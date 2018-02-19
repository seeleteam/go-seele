/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package hexutil

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func Test_Hex(t *testing.T) {
	str := "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"

	bytes, err := HexToBytes(str)
	if err != nil {
		t.Error(err.Error())
	}

	res := BytesToHex(bytes)

	assert.Equal(t, res, str)

	bytes, err = HexToBytes("")
	assert.Equal(t, err, ErrEmptyString)
}
