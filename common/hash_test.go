/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Hash(t *testing.T) {
	bytes := []byte{21}

	hash := BytesToHash(bytes)

	var exp Hash
	exp[0] = 21

	assert.Equal(t, exp, hash)
}

func Test_StringHash(t *testing.T) {
	str := "5aaeb6053f3e94c9b9a09f33669435e7"
	hash := StringToHash(str)
	res := hash.String()

	assert.Equal(t, str, res)
}

func Test_Hash_Equal(t *testing.T) {
	hash1 := StringToHash("5aaeb6053f3e94c9b9a09f33669435e7")
	hash2 := StringToHash("5aaeb6053f3e94c9b9a09f33669435e7")
	hash3 := StringToHash("5aaeb6053f3e94c9b9a09f33669435e8")

	assert.Equal(t, true, hash1.Equal(hash2))
	assert.Equal(t, false, hash1.Equal(hash3))
}
