/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"os"
	"path/filepath"
)

var (
	tempFolder string
)

func init() {
	tempFolder = filepath.Join(os.TempDir(), "SeeleTemp")
}

// GetTempFolder use a getter to implement readonly
func GetTempFolder() string {
	return tempFolder
}
