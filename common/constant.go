/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"os"
	"path/filepath"
	"os/user"
)

var (
	// tempFolder used to store temp file, such as log files
	tempFolder string

	// defaultDataFolder used to store persist data info. such as database and keystore
	defaultDataFolder string
)

func init() {
	tempFolder = filepath.Join(os.TempDir(), "SeeleTemp")

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	defaultDataFolder = filepath.Join(usr.HomeDir, ".seele")
}

// GetTempFolder use a getter to implement readonly
func GetTempFolder() string {
	return tempFolder
}

func GetDefaultDataFolder() string {
	return defaultDataFolder
}

