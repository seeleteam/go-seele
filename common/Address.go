/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"fmt"
	"log"

	"github.com/seeleteam/go-seele/common/hexutil"
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

func HexToAddress(id string) Address {
	byte, err := hexutil.HexToBytes(id)
	if err != nil {
		log.Fatal(err.Error())
	}

	nid, err := NewAddress(byte)
	if err != nil {
		log.Fatal(err.Error())
	}

	return nid
}
