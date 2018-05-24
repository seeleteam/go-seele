/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/common"
)

// Signature is a wrapper for the signed message and it is serializable.
type Signature struct {
	R *big.Int // Signature of elliptic curve cryptography.
	S *big.Int // Signature of elliptic curve cryptography.
}

// NewSignature signs the specified hash with private key and returns a signature.
// Panic if failed to sign the hash.
func NewSignature(privKey *ecdsa.PrivateKey, hash []byte) *Signature {
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash)
	if err != nil {
		panic(fmt.Errorf("Failed to sign hash, private key = %+v, hash = %v, error = %v", privKey, hash, err.Error()))
	}

	return &Signature{r, s}
}

// Verify verifies the signature against the specified hash.
// Return true if the signature is valid, otherwise false.
func (sig *Signature) Verify(signerAddress *common.Address, hash []byte) bool {
	pubKey := ToECDSAPub(signerAddress.Bytes())
	return ecdsa.Verify(pubKey, hash, sig.R, sig.S)
}
