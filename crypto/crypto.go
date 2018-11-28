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
	"github.com/seeleteam/go-seele/crypto/sha3"
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
func PubkeyToString(pub *ecdsa.PublicKey) string {
	return GetAddress(pub).Hex()
}

// Keccak512 calculates and returns the Keccak512 hash of the input data.
func Keccak512(data ...[]byte) []byte {
	d := sha3.NewKeccak512()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
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

	id := GetAddress(&keypair.PublicKey)
	return id, keypair, err
}

// GetAddress gets an address from the given public key
func GetAddress(key *ecdsa.PublicKey) *common.Address {
	addr := common.PubKeyToAddress(key, MustHash)
	return &addr
}

// PubkeyToAddress add this method for istanbul BFT integration
func PubkeyToAddress(key ecdsa.PublicKey) common.Address  {
	return *GetAddress(&key)
}

// GenerateRandomAddress generates and returns a random address.
func GenerateRandomAddress() (*common.Address, error) {
	addr, _, err := GenerateKeyPair()
	return addr, err
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
	addr, _ := MustGenerateShardKeyPair(shardNum)
	return addr
}

// MustGenerateShardKeyPair generates and returns a random address and key.
// Panic on any error.
func MustGenerateShardKeyPair(shard uint) (*common.Address, *ecdsa.PrivateKey) {
	if shard == 0 || shard > common.ShardCount {
		panic(fmt.Errorf("invalid shard number, should be between 1 and %v", common.ShardCount))
	}

	for i := 1; ; i++ {
		addr, privateKey, err := GenerateKeyPair()
		if err != nil {
			panic(err)
		}

		if addr.Shard() == shard {
			return addr, privateKey
		}
	}
}

// MustGenerateKeyPairNotShard generates and returns a random address and key.
// Panic on any error.
func MustGenerateKeyPairNotShard(shard uint) (*common.Address, *ecdsa.PrivateKey) {
	if shard == 0 || shard > common.ShardCount {
		panic(fmt.Errorf("invalid shard number, should be between 1 and %v", common.ShardCount))
	}

	for i := 1; i < 100; i++ {
		addr, privateKey, err := GenerateKeyPair()
		if err != nil {
			panic(err)
		}

		if addr.Shard() != shard {
			return addr, privateKey
		}
	}

	// panic if not generate shard after 100 times
	panic("didn't generate key pair after 100 times trying")
}

// CreateAddress creates a new address with the specified address and nonce.
// Generally, it's used to create a new contract address based on the account
// address and nonce. Note, the new created contract address and the account
// address are in the same shard.
func CreateAddress(addr common.Address, nonce uint64) common.Address {
	hash := MustHash([]interface{}{addr, nonce})
	return addr.CreateContractAddressWithHash(hash)
}

// CreateAddress2 creates an ethereum address given the address bytes, initial
// contract code hash and a salt.
func CreateAddress2(b common.Address, salt common.Hash, inithash []byte) common.Address {
	hash := Keccak256Hash([]byte{0xff}, b.Bytes(), salt.Bytes(), inithash)
	return b.CreateContractAddressWithHash(hash)
}
