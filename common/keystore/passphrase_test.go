/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package keystore

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/crypto"
)

func Test_PassPhrase(t *testing.T) {
	addr, privateKey, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(err)
	}

	password := "test"
	key := &Key{
		*addr,
		privateKey,
	}

	result, err := EncryptKey(key, password)
	assert.Equal(t, err, nil)

	decryptKey, err := DecryptKey(result, password)
	assert.Equal(t, err, nil)
	assert.Equal(t, key.Address, decryptKey.Address)
	assert.Equal(t, key.PrivateKey, decryptKey.PrivateKey)

	_, err = DecryptKey(result, "badpass")
	assert.Equal(t, err, ErrDecrypt)
}
