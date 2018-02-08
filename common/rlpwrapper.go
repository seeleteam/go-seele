/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"github.com/ethereum/go-ethereum/rlp"
)

// rlp is an effective serialize and deserialize function with no schema
// we use it as our network byte array converter

// Deserialize wrapper decode
func Deserialize(data []byte, value interface{}) error {
	return rlp.DecodeBytes(data, value)
}

// Serialize wrapper encode
func Serialize(in interface{}) ([]byte, error) {
	return rlp.EncodeToBytes(in)
}
