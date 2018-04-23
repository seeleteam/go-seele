/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package keystore

import (
	"crypto/ecdsa"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/seeleteam/go-seele/crypto"
)

// Key private key info for wallet
type Key struct {
	PrivateKey *ecdsa.PrivateKey
}

// GetKey get private key from a file
func GetKey(fileName string) (*Key, error) {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	privateKey, err := crypto.ToECDSA(content)
	if err != nil {
		return nil, err
	}

	key := &Key{
		PrivateKey: privateKey,
	}

	return key, nil
}

// StoreKey store private key in a file. Note it is not encrypted. Need to support it later.
func StoreKey(fileName string, key *Key) error {
	content := crypto.FromECDSA(key.PrivateKey)

	return writeKeyFile(fileName, content)
}

func writeKeyFile(file string, content []byte) error {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}

	// Atomic write: create a temporary hidden file first then move it into place.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}
