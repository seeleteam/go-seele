/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package crypto

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto/sha3"
)

const (
	hashLength = 32
)

// keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func keccak256Hash(data ...[]byte) []byte {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}

	h := make([]byte, hashLength)
	d.Sum(h[:0])
	return h
}

// HashBytes returns the hash of the input data.
func HashBytes(data ...[]byte) common.Hash {
	return common.BytesToHash(keccak256Hash(data...))
}

// MustHash returns the hash of the specified value.
// Panic on any error, e.g. unsupported data type for encoding.
func MustHash(v interface{}) common.Hash {
	return HashBytes(common.SerializePanic(v))
}
