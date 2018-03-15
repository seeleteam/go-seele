/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"os"
)

// IsFileOrFolderExist check if the file or folder exist
func IsFileOrFolderExist(fileOrFolder string) bool {
	_, err := os.Stat(fileOrFolder)
	return !os.IsNotExist(err)
}
