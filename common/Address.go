/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/common/hexutil"
)

const (
	addressLen = 32 // length in bytes

	addressTypeExternal = byte(1)
	addressTypeContract = byte(2)

	// Address format: version(1) + type(1) + hash[12:] + misc(10)
	// - external account misc: hash[:10]
	// - contract account misc: shardMod(2) + hash[:8]
	addressVersion1 = byte(1)
)

// EmptyAddress presents an empty address
var EmptyAddress = Address{}

// Address we use public key as node id
type Address [addressLen]byte

// NewAddress converts a byte slice to a Address
func NewAddress(b []byte) (Address, error) {
	// Validate length
	if len(b) != addressLen {
		return EmptyAddress, fmt.Errorf("wrong length, want %d bytes", addressLen)
	}

	// Validate address version
	if b[0] != addressVersion1 {
		return EmptyAddress, fmt.Errorf("invalid address version %v, expected version is %v", b[0], addressVersion1)
	}

	// Validate address type
	switch b[1] {
	case addressTypeExternal:
	case addressTypeContract:
	default:
		return EmptyAddress, fmt.Errorf("invalid address type %v", b[1])
	}

	var id Address
	copy(id[:], b)

	return id, nil
}

// PubKeyToAddress converts a ECC public key to a address.
func PubKeyToAddress(pubKey *ecdsa.PublicKey) Address {
	buf := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)

	addr, err := NewAddress(buf[1:])
	if err != nil {
		panic(err)
	}

	return addr
}

// Bytes get the actual bytes
func (id Address) Bytes() []byte {
	return id[:]
}

func (id Address) ToHex() string {
	return hexutil.BytesToHex(id.Bytes())
}

func (id Address) Equal(b Address) bool {
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

// Shard returns the shard number of address.
func (id Address) Shard() uint {
	var sum uint

	// sum [2, 21]
	for _, b := range id[2:22] {
		sum += uint(b)
	}

	// sum [22, 23] for contract address
	if id[1] == addressTypeContract {
		sum += uint(binary.BigEndian.Uint16(id[22:24]))
	}

	return (sum % ShardNumber) + 1
}
