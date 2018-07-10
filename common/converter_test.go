/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"bytes"
	"testing"

	"github.com/magiconair/properties/assert"
)

const (
	MaxUint = ^uint(0)
	MaxInt  = int(MaxUint >> 1)
)

func Test_ConvertInt64ToBytes(t *testing.T) {
	var num int64
	var numBytes []byte
	numBytes = make([]byte, 8, 8)

	num = 0
	numBytes = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	result := ConvertInt64ToBytes(num)
	assert.Equal(t, bytes.Compare(numBytes, result), 0)

	num = -1
	numBytes = []byte{255, 255, 255, 255, 255, 255, 255, 255}
	result = ConvertInt64ToBytes(num)
	assert.Equal(t, bytes.Compare(numBytes, result), 0)

	num = 100
	numBytes = []byte{0, 0, 0, 0, 0, 0, 0, 100}
	result = ConvertInt64ToBytes(num)
	assert.Equal(t, bytes.Compare(numBytes, result), 0)

	num = int64(MaxUint >> 1) // 9223372036854775807
	numBytes = []byte{127, 255, 255, 255, 255, 255, 255, 255}
	result = ConvertInt64ToBytes(num)
	assert.Equal(t, bytes.Compare(numBytes, result), 0)

	num = int64(-MaxInt - 1) // -9223372036854775808
	numBytes = []byte{128, 0, 0, 0, 0, 0, 0, 0}
	result = ConvertInt64ToBytes(num)
	assert.Equal(t, bytes.Compare(numBytes, result), 0)
}
