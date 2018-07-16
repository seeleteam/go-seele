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
	src = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	dest = CopyBytes(src)
	assert.Equal(t, bytes.Compare(src, dest), 0)

	// src is nil
	src = nil
	dest = CopyBytes(src)
	assert.Equal(t, dest, []byte(nil))
}

func Test_MustNewCache(t *testing.T) {
	cache := MustNewCache(3)
	if cache == nil {
		t.Fatal()
	}

	assert.Panic(t, func() { MustNewCache(0) }, "Must provide a positive size")
	assert.Panic(t, func() { MustNewCache(-1) }, "Must provide a positive size")
}
