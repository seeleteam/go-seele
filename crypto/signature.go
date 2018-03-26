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

// Signature is a wrapper for signed message, and is serializable.
type Signature struct {
	R *big.Int // Signature of elliptic curve cryptography.
	S *big.Int // Signature of elliptic curve cryptography.
}

// NewSignature sign the specified hash with private key and returns a signature.
// Panics if failed to sign hash.
func NewSignature(privKey *ecdsa.PrivateKey, hash []byte) *Signature {
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash)
	if err != nil {
		panic(fmt.Errorf("Failed to sign hash, private key = %+v, hash = %v, error = %v", privKey, hash, err.Error()))
	}

	return &Signature{r, s}
}

// Verify verifies the signature against the specified hash.
// Return true if signature is valid, otherwise false.
func (sig *Signature) Verify(signerAddress *common.Address, hash []byte) bool {
	pubKey := ToECDSAPub(signerAddress.Bytes())
	return ecdsa.Verify(pubKey, hash, sig.R, sig.S)
}
