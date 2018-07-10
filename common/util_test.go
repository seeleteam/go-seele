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

func Test_CopyBytes(t *testing.T) {
	// src is valid with length > 0
	src := make([]byte, 1, 1)
	dest := CopyBytes(src)
	assert.Equal(t, bytes.Compare(src, dest), 0)

	src = make([]byte, 10, 10)
	dest = CopyBytes(src)
	assert.Equal(t, bytes.Compare(src, dest), 0)

	// src is nil
	src = nil
	dest = CopyBytes(src)
	assert.Equal(t, dest, []byte(nil))
}
