/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Hash(t *testing.T) {
	bytes := []byte{21}

	hash := BytesToHash(bytes)

	var exp Hash
	exp[HashLength-1] = 21

	assert.Equal(t, exp, hash)
}

func Test_StringHash(t *testing.T) {
	str := "5aaeb6053f3e94c9b9a09f33669435e7"
	hash := StringToHash(str)
	res := string(hash.Bytes())

	assert.Equal(t, str, res)
}

func Test_Hash_Equal(t *testing.T) {
	hash1 := StringToHash("5aaeb6053f3e94c9b9a09f33669435e7")
	hash2 := StringToHash("5aaeb6053f3e94c9b9a09f33669435e7")
	hash3 := StringToHash("5aaeb6053f3e94c9b9a09f33669435e8")

	assert.Equal(t, true, hash1.Equal(hash2))
	assert.Equal(t, false, hash1.Equal(hash3))
}

func Test_ToHex(t *testing.T) {
	str := "5aaeb6053f3e94c9b9a09f33669435e7"
	hash := StringToHash(str)

	assert.Equal(t, hash.Hex(), "0x3561616562363035336633653934633962396130396633333636393433356537")
}

func Test_IsEmpty(t *testing.T) {
	hash := Hash{}
	assert.Equal(t, hash.IsEmpty(), true)

	str := "0x1"
	hash = StringToHash(str)
	assert.Equal(t, hash.IsEmpty(), false)
}

func Test_MarshalAndUnmarshalText(t *testing.T) {
	str := "0x1"
	hash := StringToHash(str)

	bytes, _ := hash.MarshalText()
	hash.UnmarshalText(bytes)

	assert.Equal(t, hash.Equal(StringToHash(str)), true)

	buff, err := json.Marshal(hash)
	assert.Equal(t, err, nil)

	var result Hash
	err = json.Unmarshal(buff, &result)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.Bytes(), hash.Bytes())
}
