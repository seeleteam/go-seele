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
