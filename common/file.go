/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"os"
)

// FileOrFolderExists checks if a file or folder exists
func FileOrFolderExists(fileOrFolder string) bool {
	_, err := os.Stat(fileOrFolder)
	return !os.IsNotExist(err)
}
