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
		return EmptyAddress, fmt.Errorf("invalid address length %v, expected length is %v", len(b), addressLen)
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

// PubKeyToAddress converts a ECC public key to an external address.
func PubKeyToAddress(pubKey *ecdsa.PublicKey, hashFunc func(interface{}) Hash) Address {
	buf := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	hash := hashFunc(buf[1:]).Bytes()

	// Address format: version(1) + type(1) + pubKeyHash[12:] + pubKeyHash[:10]
	var addr Address
	addr[0] = addressVersion1
	addr[1] = addressTypeExternal
	copy(addr[2:22], hash[12:]) // public key info
	copy(addr[22:], hash[:10])  // misc

	return addr
}

// Bytes get the actual bytes
func (id Address) Bytes() []byte {
	return id[:]
}

// ToHex converts address to 0x prefixed HEX format.
func (id Address) ToHex() string {
	return hexutil.BytesToHex(id.Bytes())
}

// Equal checks if this address is the same with the specified address b.
func (id Address) Equal(b Address) bool {
	return bytes.Equal(id[:], b[:])
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
func (id Address) Big() *big.Int { return new(big.Int).SetBytes(id[:]) }

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
func (id Address) Shard() uint {
	if IsShardDisabled {
		return 0
	}

	var sum uint

	// sum [2:22]
	for _, b := range id[2:22] {
		sum += uint(b)
	}

	// sum [22:24] for contract address
	if id[1] == addressTypeContract {
		sum += uint(binary.BigEndian.Uint16(id[22:24]))
	}

	return (sum % ShardNumber) + 1
}

// CreateContractAddress returns a contract address that in the same shard of this address.
func (id Address) CreateContractAddress(nonce uint64, hashFunc func(interface{}) Hash) Address {
	hash := hashFunc([]interface{}{id, nonce}).Bytes()

	targetShardNum := id.Shard()
	var sum uint

	// sum [12:] of public key hash
	for _, b := range hash[12:] {
		sum += uint(b)
	}

	// sum [22:24] for shard mod
	shardNum := (sum % ShardNumber) + 1
	encoded := make([]byte, 2)
	if shardNum <= targetShardNum {
		binary.BigEndian.PutUint16(encoded, uint16(targetShardNum-shardNum))
	} else {
		binary.BigEndian.PutUint16(encoded, uint16(ShardNumber+targetShardNum-shardNum))
	}

	// Address format: version(1) + type(1) + pubKeyHash[12:] + shardMod(2) + pubKeyHash[:8]
	var contractAddr Address
	contractAddr[0] = addressVersion1
	contractAddr[1] = addressTypeContract
	copy(contractAddr[2:22], hash[12:]) // public key info
	copy(contractAddr[22:24], encoded)  // shard mod
	copy(contractAddr[24:], hash[:8])   // misc

	return contractAddr
}
