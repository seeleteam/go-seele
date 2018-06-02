/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"

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
func MustSign(privKey *ecdsa.PrivateKey, hash common.Hash) *Signature {
	secKey := math.PaddedBigBytes(privKey.D, privKey.Params().BitSize/8)
	defer func(bytes []byte) {
		for i := range bytes {
			bytes[i] = 0
		}
	}(secKey)

	sig, err := secp256k1.Sign(hash.Bytes(), secKey)
	if err != nil {
		// Should never panic since the input data has correct length.
		panic(err)
	}

	return &Signature{sig}
}

// Verify verifies the signature against the specified hash.
// Return true if the signature is valid, otherwise false.
func (s Signature) Verify(signer common.Address, hash common.Hash) bool {
	if len(s.Sig) != 65 {
		return false
	}

	msg := hash.Bytes()

	pubKey, err := s.recoverPubKey(msg)
	if err != nil {
		return false // Signature was modified
	}

	if !common.PubKeyToAddress(pubKey, MustHash).Equal(signer) {
		return false
	}

	compressed := secp256k1.CompressPubkey(pubKey.X, pubKey.Y)
	return secp256k1.VerifySignature(compressed, msg, s.Sig[:64])
}

func (s Signature) recoverPubKey(msg []byte) (*ecdsa.PublicKey, error) {
	pubKey, err := secp256k1.RecoverPubkey(msg, s.Sig)
	if err != nil {
		return nil, err
	}

	curve := secp256k1.S256()
	x, y := elliptic.Unmarshal(curve, pubKey)
	return &ecdsa.PublicKey{curve, x, y}, nil
}
