/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package crypto

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto/sha3"
)

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) (h common.Hash) {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}

	d.Sum(h[:0])
	return h
}
