/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"encoding/json"
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

func Test_JsonMarshal(t *testing.T) {
	a := "0x1826603c48b4460a90af24f2d0c549b022f5a17a8f50a4a448d20ba579d01781efd18ad6b2fb90fe81207338fb0b0d6c1b6012df19c087cd8bb0e255e0c1711e"
	addr := HexMustToAddres(a)

	buff, err := json.Marshal(addr)
	assert.Equal(t, err, nil)

	var result Address
	err = json.Unmarshal(buff, &result)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.Bytes(), addr.Bytes())
}
