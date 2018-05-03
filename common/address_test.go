/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func Test_BytesToAddress(t *testing.T) {
	// Create address with single byte.
	b1 := make([]byte, 64)
	b1[63] = 1
	assert.Equal(t, BytesToAddress([]byte{1}).Bytes(), b1)

	// Create address with multiple bytes.
	b2 := make([]byte, 64)
	b2[62] = 1
	b2[63] = 2
	assert.Equal(t, BytesToAddress([]byte{1, 2}).Bytes(), b2)

	// Create address with too long bytes.
	b3 := make([]byte, 65)
	for i := 0; i < len(b3); i++ {
		b3[i] = byte(i + 1)
	}
	assert.Equal(t, BytesToAddress(b3).Bytes(), b3[1:])
}
