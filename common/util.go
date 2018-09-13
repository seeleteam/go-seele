/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"math/rand"
	"reflect"

	"github.com/hashicorp/golang-lru"
	"github.com/seeleteam/go-seele/common/hexutil"
)

// Bytes is a array byte that is converted to hex string display format when marshal
type Bytes []byte

// MarshalText implement the TextMarshaler interface
func (b Bytes) MarshalText() ([]byte, error) {
	if len(b) == 0 {
		return nil, nil
	}

	hex := hexutil.BytesToHex(b)
	return []byte(hex), nil
}

// UnmarshalText implement the TextUnmarshaler interface
func (b *Bytes) UnmarshalText(hex []byte) error {
	if len(hex) == 0 {
		return nil
	}

	arrayByte, err := hexutil.HexToBytes(string(hex))
	if err != nil {
		return err
	}

	*b = arrayByte
	return nil
}

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

// Shuffle shuffles items in slice
func Shuffle(slice interface{}) {
	rv := reflect.ValueOf(slice)
	swap := reflect.Swapper(slice)
	length := rv.Len()
	for i := length - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		swap(i, j)
	}
}
