/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"os"
	"os/user"
	"path/filepath"
)

var (
	// tempFolder used to store temp file, such as log files
	tempFolder string

	// defaultDataFolder used to store persistent data info, such as the database and keystore
	defaultDataFolder string

	// PrintLog default is false. If it is true, all logs will be printed in the console, otherwise they will be stored in the file.
	PrintLog = true

	// IsDebug default is false. If IsDebug is true, the log level will be DebugLevel, otherwise it is InfoLevel
	IsDebug = false
)

func init() {
	tempFolder = filepath.Join(os.TempDir(), "SeeleTemp")

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	defaultDataFolder = filepath.Join(usr.HomeDir, ".seele")
}

// GetTempFolder uses a getter to implement readonly
func GetTempFolder() string {
	return tempFolder
}

// GetDefaultDataFolder gets the default data Folder
func GetDefaultDataFolder() string {
	return defaultDataFolder
}
