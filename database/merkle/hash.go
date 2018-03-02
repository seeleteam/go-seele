/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package merkle

import (
	"github.com/seeleteam/go-seele/crypto"
)

func hashBytes(value []byte) []byte {
	return crypto.Keccak256Hash(value)
}
