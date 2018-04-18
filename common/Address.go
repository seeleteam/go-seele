/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"fmt"

	"github.com/seeleteam/go-seele/common/hexutil"
	"bytes"
)

const (
	addressIDBits = 512 // the length of the public key
)

// Address we use public key as node id
type Address [addressIDBits / 8]byte

// NewAddress converts a byte slice to a Address
func NewAddress(b []byte) (Address, error) {
	var id Address
	if len(b) != len(id) {
		return id, fmt.Errorf("wrong length, want %d bytes", len(id))
	}
	copy(id[:], b)
	return id, nil
}

// Bytes get the actual bytes
func (id *Address) Bytes() []byte {
	return id[:]
}

func (id *Address) ToHex() string {
	return hexutil.BytesToHex(id.Bytes())
}

func (id *Address) Equal(b Address) bool {
	return bytes.Equal(id[:], b[:])
}

func HexToAddress(id string) (Address, error) {
	byte, err := hexutil.HexToBytes(id)
	if err != nil {
		return Address{}, err
	}

	nid, err := NewAddress(byte)
	if err != nil {
		return Address{}, err
	}

	return nid, nil
}

func HexMustToAddres(id string) Address {
	a, err := HexToAddress(id)
	if err != nil {
		panic(err)
	}

	return a
}
