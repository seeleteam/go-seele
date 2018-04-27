/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"os"
)

// DoesFileOrFolderExist checks if the file or folder exists
func DoesFileOrFolderExist(fileOrFolder string) bool {
	_, err := os.Stat(fileOrFolder)
	return !os.IsNotExist(err)
}
