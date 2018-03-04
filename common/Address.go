/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"fmt"
	"log"

	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/crypto"
)

const (
	addressIDBits = 512 // the length of the public key
	// AccAddressLengh account address length
	AccAddressLengh = 20
)

// Address we use public key as node id
type Address [addressIDBits / 8]byte

// AccAddress account address used as id of account
type AccAddress [AccAddressLengh]byte

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

// ToSha get the node hash
func (id *Address) ToSha() *Hash {
	data := crypto.Keccak256Hash(id[:])
	h := BytesToHash(data)
	return &h
}

func GenerateRandomAddress() (*Address, error) {
	keypair, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	buff := crypto.FromECDSAPub(&keypair.PublicKey)

	id, err := NewAddress(buff[1:])
	if err != nil {
		return nil, err
	}

	return &id, err
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
