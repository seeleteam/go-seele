/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"bytes"
	"fmt"
	"math/big"

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
func (id Address) Bytes() []byte {
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

// BytesToAddress converts the specified byte array to Address.
func BytesToAddress(bs []byte) Address {
	var addr Address

	if len(bs) > len(addr) {
		bs = bs[len(bs)-len(addr):]
	}

	copy(addr[len(addr)-len(bs):], bs)

	return addr
}

// BigToAddress converts a big int to address.
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// Big converts address to a big int.
func (id Address) Big() *big.Int { return new(big.Int).SetBytes(id[:]) }

func (id Address) MarshalText() ([]byte, error) {
	str := id.ToHex()
	return []byte(str), nil
}

func (id *Address) UnmarshalText(json []byte) error {
	a, err := HexToAddress(string(json))
	if err != nil {
		return err
	}

	copy(id[:], a[:])
	return nil
}
