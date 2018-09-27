/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"fmt"
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
	// Create the file directory with appropriate permissions
	// in case it is not present yet.
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// Atomic write: create a temporary hidden file first then move it into place.
	f, err := ioutil.TempFile(filepath.Dir(filePath), fmt.Sprint(".", filepath.Base(filePath), ".tmp"))
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
