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

	// defaultDataFolder used to store persist data info. such as database and keystore
	defaultDataFolder string

	// PrintLog default is false. If it is true, it will not print all the logs in the console. otherwise, will write log in file.
	PrintLog = false

	// IsDebug default is false. If it is true, the log level is set to DebugLevel. otherwise, the log level is set to InfoLevel
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

// GetTempFolder use a getter to implement readonly
func GetTempFolder() string {
	return tempFolder
}

func GetDefaultDataFolder() string {
	return defaultDataFolder
}
