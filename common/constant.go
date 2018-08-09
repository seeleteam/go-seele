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
)

const (
	// ShardCount represents the total number of shards.
	ShardCount = 20

	// PrintExplosionLog whether print explosion log flag. Most of them are transaction track logs
	PrintExplosionLog = false

	// MetricsRefreshTime is the time of metrics sleep 1 minute
	MetricsRefreshTime = time.Minute

	// CPUMetricsRefreshTime is the time of metrics monitor cpu
	CPUMetricsRefreshTime = time.Second
)

var (
	// tempFolder used to store temp file, such as log files
	tempFolder string

	// defaultDataFolder used to store persistent data info, such as the database and keystore
	defaultDataFolder string
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
