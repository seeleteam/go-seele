/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package keystore

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/crypto"
)

func Test_KeyStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "keystore")
	if err != nil {
		panic(err)
	}

	addr, keypair, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(err)
	}

	key := &Key{
		*addr,
		keypair,
	}

	password := "testfile"
	fileName := filepath.Join(dir, "keyfile")
	err = StoreKey(fileName, password, key)
	assert.Equal(t, err, nil)

	result, err := GetKey(fileName, password)
	assert.Equal(t, err, nil)
	assert.Equal(t, crypto.FromECDSA(key.PrivateKey), crypto.FromECDSA(result.PrivateKey))
	assert.Equal(t, key.Address, result.Address)
}
