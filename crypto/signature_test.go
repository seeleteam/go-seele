/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package crypto

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func Test_Sign(t *testing.T) {
	privKey, err := GenerateKey()
	assert.Equal(t, err, nil)

	// Sign successfully
	hash := MustHash("test message")
	signature := MustSign(privKey, hash.Bytes())
	assert.Equal(t, len(signature.Sig), 65)

	// Succeed to verify signature.
	signer := GetAddress(&privKey.PublicKey)
	assert.Equal(t, signature.Verify(*signer, hash.Bytes()), true)

	// Failed to verify signature if msg changed.
	hash2 := MustHash("test message 2")
	assert.Equal(t, signature.Verify(*signer, hash2.Bytes()), false)

	// Failed to verify signature if signer changed.
	privKey2, _ := GenerateKey()
	signer2 := GetAddress(&privKey2.PublicKey)
	assert.Equal(t, signature.Verify(*signer2, hash.Bytes()), false)
}
