/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/crypto/secp256k1"
)

const (
	ecdsaPublickKeyPrefix byte = 4
)

// GenerateKey generates and returns an ECDSA private key.
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(S256(), rand.Reader)
}

// ToECDSAPub creates ecdsa.PublicKey object by the given byte array.
// Pubkey length derived from ecdsa is 65, with constant prefix 4 in the first byte.
// So if pubkey length equals 64, we insert one byte in front
func ToECDSAPub(pub []byte) *ecdsa.PublicKey {
	if len(pub) == 0 {
		return nil
	}

	var pubNew []byte

	if len(pub) == 65 {
		pubNew = pub
	} else {
		pubNew = make([]byte, 65)
		pubNew[0] = ecdsaPublickKeyPrefix
		copy(pubNew[1:], pub[0:])
	}
	x, y := elliptic.Unmarshal(S256(), pubNew)
	return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}
}

// PubkeyToString returns the string of the given public key, with prefix 0x
func PubkeyToString(pub *ecdsa.PublicKey) (pubStr string) {
	buff := FromECDSAPub(pub)
	pubStr = "0x" + hex.EncodeToString(buff[1:])
	return
}

// FromECDSAPub marshals and returns the byte array of the specified ECDSA public key.
func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(pub.Curve, pub.X, pub.Y)
}

// S256 returns an instance of the secp256k1 curve.
func S256() elliptic.Curve {
	return secp256k1.S256()
}

// LoadECDSAFromString creates ecdsa private key from the given string.
// ecStr should start with 0x or 0X
func LoadECDSAFromString(ecStr string) (*ecdsa.PrivateKey, error) {
	if !hexutil.Has0xPrefix(ecStr) {
		return nil, errors.New("Input string not a valid ecdsa string")
	}
	key, err := hex.DecodeString(ecStr[2:])
	if err != nil {
		return nil, err
	}
	return ToECDSA(key)
}

// ToECDSA creates a private key with the given D value.
func ToECDSA(d []byte) (*ecdsa.PrivateKey, error) {
	return toECDSA(d, true)
}

// toECDSA creates a private key with the given D value. The strict parameter
// controls whether the key's length should be enforced at the curve size or
// it can also accept legacy encodings (0 prefixes).
func toECDSA(d []byte, strict bool) (*ecdsa.PrivateKey, error) {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = S256()
	if strict && 8*len(d) != priv.Params().BitSize {
		return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
	}
	priv.D = new(big.Int).SetBytes(d)
	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	return priv, nil
}

// FromECDSA exports a private key into a binary dump.
func FromECDSA(priv *ecdsa.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	return math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
}

// GenerateKeyPair generates public key and private key
func GenerateKeyPair() (*common.Address, *ecdsa.PrivateKey, error) {
	keypair, err := GenerateKey()
	if err != nil {
		return nil, nil, err
	}

	id, err := GetAddress(keypair)
	if err != nil {
		return nil, nil, err
	}

	return id, keypair, err
}

// GetAddress gets an address from the given private key
func GetAddress(key *ecdsa.PrivateKey) (*common.Address, error) {
	buff := FromECDSAPub(&key.PublicKey)
	id, err := common.NewAddress(buff[1:])
	if err != nil {
		return nil, err
	}

	return &id, nil
}

// MustGetAddress gets an address from the given private key. in fact the error will not occur, so MustGetAddress will always work
func MustGetAddress(key *ecdsa.PrivateKey) *common.Address {
	addr, err := GetAddress(key)
	if err != nil {
		panic(err)
	}

	return addr
}

// GenerateRandomAddress generates and returns a random address.
func GenerateRandomAddress() (*common.Address, error) {
	publicKey, _, error := GenerateKeyPair()

	return publicKey, error
}

// MustGenerateRandomAddress generates and returns a random address.
// Panic on any error.
func MustGenerateRandomAddress() *common.Address {
	address, err := GenerateRandomAddress()
	if err != nil {
		panic(err)
	}

	return address
}

// MustGenerateShardAddress generates and returns a random address that match the specified shard number.
// Panic on any error.
func MustGenerateShardAddress(shardNum uint) *common.Address {
	if shardNum == 0 || shardNum > common.ShardNumber {
		panic(fmt.Errorf("invalid shard number, should be between 1 and %v", common.ShardNumber))
	}

	for {
		address, err := GenerateRandomAddress()
		if err != nil {
			panic(err)
		}

		if common.GetShardNumber(*address) == shardNum {
			return address
		}
	}
}

// CreateAddress creates a new address with the specified address and nonce.
// Generally, it's used to create a new contract address based on the account
// address and nonce. Note, the new created contract address and the account
// address are in the same shard.
func CreateAddress(addr common.Address, nonce uint64) common.Address {
	addrHash := MustHash(addr)
	nonceHash := MustHash(nonce)

	return common.CreateContractAddress(addr, addrHash.Bytes(), nonceHash.Bytes())
}
