/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package hexutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	// ErrInvalidOddLength
	str = "0x5aaeb6053f3e94c9b9a09f3"
	bytes, err = HexToBytes(str)
	assert.Equal(t, err, ErrInvalidOddLength)
}

func Test_Has0xPrefix(t *testing.T) {
	// Normal case 1
	str := "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"
	result := Has0xPrefix(str)
	assert.Equal(t, result, true)

	// Normal case 2
	str = "0X5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"
	result = Has0xPrefix(str)
	assert.Equal(t, result, true)

	// Normal case 3
	str = "0x"
	result = Has0xPrefix(str)
	assert.Equal(t, result, true)

	// Normal case 4
	str = "0X"
	result = Has0xPrefix(str)
	assert.Equal(t, result, true)

	// Bad case 1
	str = "0"
	result = Has0xPrefix(str)
	assert.Equal(t, result, false)

	// Bad case 2
	str = "1x"
	result = Has0xPrefix(str)
	assert.Equal(t, result, false)

	// Bad case 3
	str = "1X"
	result = Has0xPrefix(str)
	assert.Equal(t, result, false)

	// Bad case 4
	str = "0a"
	result = Has0xPrefix(str)
	assert.Equal(t, result, false)
}
