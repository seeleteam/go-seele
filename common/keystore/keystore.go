/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package keystore

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// GetKey get private key from a file
func GetKey(fileName, password string) (*Key, error) {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	return DecryptKey(content, password)
}

// StoreKey store private key in a file. Note it is not encrypted. Need to support it later.
func StoreKey(fileName, password string, key *Key) error {
	content, err := EncryptKey(key, password)
	if err != nil {
		return err
	}

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
