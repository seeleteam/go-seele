/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/crypto/secp256k1"
)

// Signature is a wrapper for signed message and signer public key.
type Signature struct {
	pubKey *ecdsa.PublicKey
	r      *big.Int
	s      *big.Int
}

// NewSignature sign the specified hash with private key and returns a signature.
// Panics if failed to sign hash.
func NewSignature(privKey *ecdsa.PrivateKey, hash []byte) (*Signature, error) {
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash)
	if err != nil {
		return nil, fmt.Errorf("Failed to sign hash, private key = %+v, hash = %v, error = %v", privKey, hash, err.Error())
	}

	return &Signature{&privKey.PublicKey, r, s}, nil
}

// Verify verifies the signature against the specified hash.
// Return true if signature is valid, otherwise false.
func (sig *Signature) Verify(hash []byte) bool {
	return ecdsa.Verify(sig.pubKey, hash, sig.r, sig.s)
}

func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(S256(), rand.Reader)
}

func ToECDSAPub(pub []byte) *ecdsa.PublicKey {
	if len(pub) == 0 {
		return nil
	}
	x, y := elliptic.Unmarshal(S256(), pub)
	return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}
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
