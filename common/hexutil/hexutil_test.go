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

	// ErrEmptyString
	bytes, err = HexToBytes("")
	assert.Equal(t, err, ErrEmptyString)

	// ErrSyntax
	str = "0x78780d010387113120864842000ccbe40d0-"
	bytes, err = HexToBytes(str)
	assert.Equal(t, err, ErrSyntax)

	// ErrMissingPrefix
	str = "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"
	bytes, err = HexToBytes(str)
	assert.Equal(t, err, ErrMissingPrefix)

	// ErrOddLength
	str = "0x5aaeb6053f3e94c9b9a09f3"
	bytes, err = HexToBytes(str)
	assert.Equal(t, err, ErrOddLength)
}
