/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FileOrFolderExists(t *testing.T) {
	assert.Equal(t, FileOrFolderExists("notexist"), false)

	file := filepath.Join(os.TempDir(), "existfile")
	result, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(file)

	result.Close()

	assert.Equal(t, FileOrFolderExists(file), true)
}

func Test_SaveFile(t *testing.T) {
	file := filepath.Join(os.TempDir(), "testsavefile.json")
	assert.Equal(t, FileOrFolderExists(file), false)

	err := SaveFile(file, []byte("qq"))
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(file)

	assert.Equal(t, FileOrFolderExists(file), true)
}
