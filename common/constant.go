/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"os"
	"os/user"
	"path/filepath"

	"crypto/ecdsa"

	"github.com/seeleteam/go-seele/log/comm"
)

const ShardNumber = 20

var (
	// tempFolder used to store temp file, such as log files
	tempFolder string

	// defaultDataFolder used to store persistent data info, such as the database and keystore
	defaultDataFolder string

	// LogConfig is the Configuration of log
	LogConfig = &comm.LogConfig{PrintLog: true, IsDebug: true}
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

// GetMapOnlyValue gets the map only value
func GetMapOnlyValue(privateKey map[string]*ecdsa.PrivateKey) *ecdsa.PrivateKey {
	if len(privateKey) == 1 {
		for _, v := range privateKey {
			return v
		}
	}
	return nil
}
