/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/seeleteam/go-seele/log/comm"
)

//ShardCount represents the total number of shards.
const ShardCount = 20

// PrintExplosionLog whether print explosion log flag. Most of them are transaction track logs
const PrintExplosionLog = false

// RefreshTime is the time of metrics sleep 1 minute
const RefreshTime = time.Minute

var (
	// tempFolder used to store temp file, such as log files
	tempFolder string

	// defaultDataFolder used to store persistent data info, such as the database and keystore
	defaultDataFolder string

	// LogConfig is the Configuration of log
	LogConfig = &comm.LogConfig{PrintLog: true, IsDebug: true}

	// LogFileName default log file name
	LogFileName = "log.txt"
)

func init() {
	tempFolder = filepath.Join(os.TempDir(), "seeleTemp")

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
