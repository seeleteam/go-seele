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
	"math/big"

	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/common/hexutil"
)

//////////////////////////////////////////////////////////////////////////////
// Address format:
// - External account: pubKeyHash[12:32] and set last 4 bits to addressTypeExternal(1)
// - Contract account: AddrNonceHash[14:32] and set last 4 bits to addressTypeContract(2), the left 12 bits for shard (max shard is 4096).
//////////////////////////////////////////////////////////////////////////////

// AddressType represents the address type
type AddressType byte

const (
	addressLen = 20 // length in bytes

	// AddressTypeExternal is the address type for external account.
	AddressTypeExternal = AddressType(1)
	// AddressTypeContract is the address type for contract account.
	AddressTypeContract = AddressType(2)
	// AddressTypeReserved is the reserved address type for system contract.
	AddressTypeReserved = AddressType(3)
)

// EmptyAddress presents an empty address
var EmptyAddress = Address{}

// MaxSystemContractAddress max system contract address
var MaxSystemContractAddress = BytesToAddress([]byte{4, 255})

// Address we use public key as node id
type Address [addressLen]byte

// NewAddress converts a byte slice to a Address
func NewAddress(b []byte) (Address, error) {
	// Validate length
	if len(b) != addressLen {
		return EmptyAddress, errors.Create(errors.ErrAddressLenInvalid, len(b), addressLen)
	}

	var id Address
	copy(id[:], b)

	return id, nil
}

// PubKeyToAddress converts a ECC public key to an external address.
func PubKeyToAddress(pubKey *ecdsa.PublicKey, hashFunc func(interface{}) Hash) Address {
	buf := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	hash := hashFunc(buf[1:]).Bytes()

	var addr Address
	copy(addr[:], hash[12:]) // use last 20 bytes of public key hash

	// set address type in the last 4 bits
	addr[19] &= 0xF0
	addr[19] |= byte(AddressTypeExternal)

	return addr
}

func (id *Address) IsEVMContract() bool {
	return id.Type() == AddressTypeContract
}

// Type returns the address type
func (id *Address) Type() AddressType {
	if id.IsReserved() {
		return AddressTypeReserved
	}

	return AddressType(id[addressLen-1] & 0x0F)
}

// IsReserved returns true if the address is reserved
func (id *Address) IsReserved() bool {
	return !id.IsEmpty() && bytes.Compare(id.Bytes(), MaxSystemContractAddress.Bytes()) <= 0
}

// Bytes get the actual bytes
//
// Note: if we want to use pointer type, need to change the code snippet in unit test:
//   BytesToAddress([]byte{1, 2}).Bytes()
//   ->
//   addrBytes := BytesToAddress([]byte{1, 2})
//   (&addrBytes).Bytes()
//
// refer link: https://stackoverflow.com/questions/10535743/address-of-a-temporary-in-go
func (id Address) Bytes() []byte {
	return id[:]
}

// ToHex converts address to 0x prefixed HEX format.
func (id *Address) ToHex() string {
	return hexutil.BytesToHex(id.Bytes())
}

// Equal checks if this address is the same with the specified address b.
func (id *Address) Equal(b Address) bool {
	return bytes.Equal(id[:], b[:])
}

// IsEmpty returns true if this address is empty. Otherwise, false.
func (id *Address) IsEmpty() bool {
	return id.Equal(EmptyAddress)
}

// HexToAddress converts the specified HEX string to address.
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

// HexMustToAddres converts the specified HEX string to address.
// Panics on any error.
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
func (id *Address) Big() *big.Int { return new(big.Int).SetBytes(id[:]) }

// MarshalText marshals the address to HEX string.
func (id Address) MarshalText() ([]byte, error) {
	str := id.ToHex()
	return []byte(str), nil
}

// UnmarshalText unmarshals address from HEX string.
func (id *Address) UnmarshalText(json []byte) error {
	a, err := HexToAddress(string(json))
	if err != nil {
		return err
	}

	copy(id[:], a[:])
	return nil
}

// Shard returns the shard number of this address.
func (id *Address) Shard() uint {
	var sum uint

	// sum [0:18]
	for _, b := range id[:18] {
		sum += uint(b)
	}

	// sum [18:20] except address type
	tail := uint(binary.BigEndian.Uint16(id[18:]))
	sum += (tail >> 4)

	return (sum % ShardCount) + 1
}

// CreateContractAddress returns a contract address that in the same shard of this address.
func (id *Address) CreateContractAddress(nonce uint64, hashFunc func(interface{}) Hash) Address {
	hash := hashFunc([]interface{}{id, nonce}).Bytes()

	targetShardNum := id.Shard()
	var sum uint

	// sum [14:] of public key hash
	for _, b := range hash[14:] {
		sum += uint(b)
	}

	// sum [18:20] for shard mod and contract address type
	shardNum := (sum % ShardCount) + 1
	encoded := make([]byte, 2)
	var mod uint
	if shardNum <= targetShardNum {
		mod = targetShardNum - shardNum
	} else {
		mod = ShardCount + targetShardNum - shardNum
	}
	mod <<= 4
	mod |= uint(AddressTypeContract) // set address type in the last 4 bits
	binary.BigEndian.PutUint16(encoded, uint16(mod))

	var contractAddr Address
	copy(contractAddr[:18], hash[14:]) // use last 18 bytes of hash (from address + nonce)
	copy(contractAddr[18:], encoded)   // last 2 bytes for shard mod and address type

	return contractAddr
}
