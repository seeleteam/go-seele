/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package keystore

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/seeleteam/go-seele/common/errors"
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
	assert.Equal(t, len(result), 376)

	decryptKey, err := DecryptKey(result, password)
	assert.Equal(t, err, nil)
	assert.Equal(t, key.Address, decryptKey.Address)
	assert.Equal(t, key.PrivateKey, decryptKey.PrivateKey)

	_, err = DecryptKey(result, "badpass")
	assert.Equal(t, err, errors.Get(errors.ErrDecrypt))

	// Empty password
	_, err = EncryptKey(key, "")
	assert.Equal(t, err, errors.Get(errors.ErrEmptyAuthKey))

	// Version not match
	var encKey encryptedKey
	err = json.Unmarshal(result, &encKey)
	encKey.Version = 2
	result, err = json.MarshalIndent(encKey, "", "\t")
	_, err = DecryptKey(result, password)
	assert.Equal(t, err.Error(), "Version not supported: 2")
}
