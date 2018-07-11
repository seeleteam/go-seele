/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"github.com/hashicorp/golang-lru"
)

// CopyBytes copies and returns a new bytes from the specified source bytes.
func CopyBytes(src []byte) []byte {
	if src == nil {
		return nil
	}

	dest := make([]byte, len(src))
	copy(dest, src)
	return dest
}

// MustNewCache creates a LRU cache with specified size. Panics on any error.
func MustNewCache(size int) *lru.Cache {
	cache, err := lru.New(size)
	if err != nil {
		panic(err) // error occurs only when size <= 0.
	}

	return cache
}
