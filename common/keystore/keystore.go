/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package keystore

import (
	"io/ioutil"

	"github.com/seeleteam/go-seele/common"
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

	return common.SaveFile(fileName, content)
}
