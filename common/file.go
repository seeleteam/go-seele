/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// FileOrFolderExists checks if a file or folder exists
func FileOrFolderExists(fileOrFolder string) bool {
	_, err := os.Stat(fileOrFolder)
	return !os.IsNotExist(err)
}

// SaveFile save file
func SaveFile(filePath string, content []byte) error {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(filePath), dirPerm); err != nil {
		return err
	}

	// Atomic write: create a temporary hidden file first then move it into place.
	f, err := ioutil.TempFile(filepath.Dir(filePath), "."+filepath.Base(filePath)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), filePath)
}
