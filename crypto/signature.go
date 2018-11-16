/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto/secp256k1"
)

// Signature is a wrapper for the signed message and it is serializable.
type Signature struct {
	Sig []byte // [R || S || V] format signature in 65 bytes.
}

// MustSign signs the specified hash with private key and returns a signature.
// Panic if failed to sign the hash.
func MustSign(privKey *ecdsa.PrivateKey, hash []byte) *Signature {
	sig, err := Sign(privKey, hash)

	if err != nil {
		panic(err)
	}

	return sig
}

// Sign signs the specified bytes with private key and returns a signature.
func Sign(privKey *ecdsa.PrivateKey, buff []byte) (*Signature, error) {
	secKey := math.PaddedBigBytes(privKey.D, privKey.Params().BitSize/8)
	defer func(bytes []byte) {
		for i := range bytes {
			bytes[i] = 0
		}
	}(secKey)

	sig, err := secp256k1.Sign(buff, secKey)
	if err != nil {
		return nil, err
	}

	return &Signature{sig}, nil
}

// Verify verifies the signature against the specified hash.
// Return true if the signature is valid, otherwise false.
func (s Signature) Verify(signer common.Address, hash []byte) bool {
	if len(s.Sig) != 65 {
		return false
	}

	pubKey, err := SigToPub(hash, s.Sig)
	if err != nil {
		return false // Signature was modified
	}

	if !GetAddress(pubKey).Equal(signer) {
		return false
	}

	compressed := secp256k1.CompressPubkey(pubKey.X, pubKey.Y)
	return secp256k1.VerifySignature(compressed, hash, s.Sig[:64])
}

func Ecrecover(hash, sig []byte) ([]byte, error) {
	return secp256k1.RecoverPubkey(hash, sig)
}

func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	s, err := Ecrecover(hash, sig)
	if err != nil {
		return nil, err
	}

	x, y := elliptic.Unmarshal(S256(), s)
	return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}, nil
}

/////////////////////////////////////////////////////////////////////////////////////////
// Below code is copied from ETH, and used to support precompiled contract 'ecrecover'.
// If import ethereum crypto pkg as vendor code, there will be compilation error about
// reference cgo code.
/////////////////////////////////////////////////////////////////////////////////////////
var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1halfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

// ValidateSignatureValues verifies whether the signature values are valid with
// the given chain rules. The v value is assumed to be either 0 or 1.
func ValidateSignatureValues(v byte, r, s *big.Int, homestead bool) bool {
	if r.Cmp(common.Big1) < 0 || s.Cmp(common.Big1) < 0 {
		return false
	}
	// reject upper range of s values (ECDSA malleability)
	// see discussion in secp256k1/libsecp256k1/include/secp256k1.h
	if homestead && s.Cmp(secp256k1halfN) > 0 {
		return false
	}
	// Frontier: allow s to be in full N range
	return r.Cmp(secp256k1N) < 0 && s.Cmp(secp256k1N) < 0 && (v == 0 || v == 1)
}
