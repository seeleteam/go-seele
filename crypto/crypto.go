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
	"github.com/seeleteam/go-seele/crypto/secp256k1"
)

const (
	ecdsaPublickKeyPrefix byte = 4
)

// Signature is a wrapper for signed message and signer public key.
type Signature struct {
	pubKey *ecdsa.PublicKey
	r      *big.Int
	s      *big.Int
}

// NewSignature sign the specified hash with private key and returns a signature.
// Panics if failed to sign hash.
func NewSignature(privKey *ecdsa.PrivateKey, hash []byte) *Signature {
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash)
	if err != nil {
		panic(fmt.Errorf("Failed to sign hash, private key = %+v, hash = %v, error = %v", privKey, hash, err.Error()))
	}

	return &Signature{&privKey.PublicKey, r, s}
}

// Verify verifies the signature against the specified hash.
// Return true if signature is valid, otherwise false.
func (sig *Signature) Verify(hash []byte) bool {
	return ecdsa.Verify(sig.pubKey, hash, sig.r, sig.s)
}

func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(S256(), rand.Reader)
}

// ToECDSAPub create ecdsa.PublicKey object by byte array.
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

// PubkeyToString returns string of public key, prefix with 0x
func PubkeyToString(pub *ecdsa.PublicKey) (pubStr string) {
	buff := FromECDSAPub(pub)
	pubStr = "0x" + hex.EncodeToString(buff[1:])
	return
}

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

// LoadECDSAFromString create ecdsa privatekey from string
// ecStr should start with 0x or 0X
func LoadECDSAFromString(ecStr string) (*ecdsa.PrivateKey, error) {
	if len(ecStr) >= 2 && ecStr[0] == '0' && (ecStr[1] == 'x' || ecStr[1] == 'X') {
		key, err := hex.DecodeString(ecStr[2:])
		if err != nil {
			return nil, err
		}
		return ToECDSA(key)
	}
	return nil, errors.New("Input string not a valid ecdsa string")
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
