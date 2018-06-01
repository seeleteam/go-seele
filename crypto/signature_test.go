/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package crypto

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
)

func Test_Sign(t *testing.T) {
	privKey, err := GenerateKey()
	assert.Equal(t, err, nil)

	// Sign successfully
	hash := MustHash("test message")
	signature := MustSign(privKey, hash)
	assert.Equal(t, len(signature.Sig), 65)

	// Succeed to verify signature.
	signer := common.PubKeyToAddress(&privKey.PublicKey)
	assert.Equal(t, signature.Verify(signer, hash), true)

	// Failed to verify signature if msg changed.
	hash2 := MustHash("test message 2")
	assert.Equal(t, signature.Verify(signer, hash2), false)

	// Failed to verify signature if signer changed.
	privKey2, _ := GenerateKey()
	signer2 := common.PubKeyToAddress(&privKey2.PublicKey)
	assert.Equal(t, signature.Verify(signer2, hash), false)
}
