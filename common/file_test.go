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

func Test_DoesFileOrFolderExist(t *testing.T) {
	assert.Equal(t, DoesFileOrFolderExist("notexist"), false)

	file := filepath.Join(os.TempDir(), "existfile")
	result, err := os.Create(file)
	if err != nil {
		panic(err)
	}

	result.Close()

	assert.Equal(t, DoesFileOrFolderExist(file), true)
}
