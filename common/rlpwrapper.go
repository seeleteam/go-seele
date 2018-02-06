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

// Decoding wrapper decode
func Decoding(data []byte, value interface{}) error {
	err := rlp.DecodeBytes(data, value)
	return err
}

// Encoding wrapper encode
func Encoding(in interface{}) ([]byte, error) {
	return rlp.EncodeToBytes(in)
}
