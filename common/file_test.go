/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/magiconair/properties/assert"
)

func Test_IsFileOrFolderExist(t *testing.T) {
	assert.Equal(t, IsFileOrFolderExist("notexist"), false)

	file := filepath.Join(os.TempDir(), "existfile")
	result, err := os.Create(file)
	if err != nil {
		panic(err)
	}

	result.Close()

	assert.Equal(t, IsFileOrFolderExist(file), true)
}
