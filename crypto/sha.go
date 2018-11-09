/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package crypto

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto/sha3"
)

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) common.Hash {
	return common.BytesToHash(Keccak256(data...))
}

// HashBytes returns the hash of the input data.
func HashBytes(data ...[]byte) common.Hash {
	return common.BytesToHash(Keccak256(data...))
}

// MustHash returns the hash of the specified value.
// Panic on any error, e.g. unsupported data type for encoding.
func MustHash(v interface{}) common.Hash {
	return HashBytes(common.SerializePanic(v))
}
