/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"os"
)

func IsFileOrFolderExist(fileOrFolder string) bool {
	if _, err := os.Stat(fileOrFolder); os.IsNotExist(err) {
		return false
	}

	return true
}