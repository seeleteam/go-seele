/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package keystore

import (
	"testing"
	"io/ioutil"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/magiconair/properties/assert"
	"path/filepath"
)

func Test_KeyStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "keystore")
	if err != nil {
		panic(err)
	}

	fileName := filepath.Join(dir, "keyfile")
	keypair, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	key := &Key{
		PrivateKey:keypair,
	}

	err = StoreKey(fileName, key)
	assert.Equal(t, err, nil)

	result, err := GetKey(fileName)
	assert.Equal(t, err, nil)
	assert.Equal(t, crypto.FromECDSA(key.PrivateKey), crypto.FromECDSA(result.PrivateKey))
}