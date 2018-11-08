/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"
)

const (

	// SeeleProtoName protoName of Seele service
	SeeleProtoName = "seele"

	// SeeleVersion Version number of Seele protocol
	SeeleVersion uint = 1

	// ShardCount represents the total number of shards.
	ShardCount = 2

	// PrintExplosionLog whether print explosion log flag. Most of them are transaction track logs
	PrintExplosionLog = false

	// MetricsRefreshTime is the time of metrics sleep 1 minute
	MetricsRefreshTime = time.Minute

	// CPUMetricsRefreshTime is the time of metrics monitor cpu
	CPUMetricsRefreshTime = time.Second

	// ConfirmedBlockNumber is the block number for confirmed a block, it should be more than 12 in product
	ConfirmedBlockNumber = 120

	// LightChainDir lightchain data directory based on config.DataRoot
	LightChainDir = "/db/lightchain"

	// EthashAlgorithm miner algorithm ethash
	EthashAlgorithm = "ethash"

	// Sha256Algorithm miner algorithm sha256
	Sha256Algorithm = "sha256"

	// EVMStackLimit increase evm stack limit to 8192
	EVMStackLimit = 8192

	WindowsPipeDir = `\\.\pipe\`

	DefaultPipeFile = `\seele.ipc`

  // BlockPackInterval it's an estimate time.
	BlockPackInterval = 15 * time.Second
)

var (
	// tempFolder used to store temp file, such as log files
	tempFolder string

	// defaultDataFolder used to store persistent data info, such as the database and keystore
	defaultDataFolder string

	// defaultIPCPath used to store the ipc file
	defaultIPCPath string
)

func init() {
	tempFolder = filepath.Join(os.TempDir(), "seeleTemp")

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	defaultDataFolder = filepath.Join(usr.HomeDir, ".seele")

	if runtime.GOOS == "windows" {
		defaultIPCPath = WindowsPipeDir + DefaultPipeFile
	} else {
		defaultIPCPath = filepath.Join(defaultDataFolder, DefaultPipeFile)
	}
}

// GetTempFolder uses a getter to implement readonly
func GetTempFolder() string {
	return tempFolder
}

// GetDefaultDataFolder gets the default data Folder
func GetDefaultDataFolder() string {
	return defaultDataFolder
}

func GetDefaultIPCPath() string {
	return defaultIPCPath
}
