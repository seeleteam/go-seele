/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"math/big"
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

	// SeeleVersion for simpler display
	SeeleNodeVersion string = "v1.2.0"
  
	// ShardCount represents the total number of shards.
	ShardCount = 4

	// MetricsRefreshTime is the time of metrics sleep 1 minute
	MetricsRefreshTime = time.Minute

	// CPUMetricsRefreshTime is the time of metrics monitor cpu
	CPUMetricsRefreshTime = time.Second

	// ConfirmedBlockNumber is the block number for confirmed a block, it should be more than 12 in product
	ConfirmedBlockNumber = 120

	// ForkHeight after this height we change the content of block: hardFork
	ForkHeight = 130000

	// ForkHeight after this height we change the content of block: hardFork
	SecondForkHeight = 145000

	// ForkHeight after this height we change the validation of tx: hardFork
	ThirdForkHeight = 735000

	SmartContractNonceForkHeight = 1100000

	// LightChainDir lightchain data directory based on config.DataRoot
	LightChainDir = "/db/lightchain"

	// EthashAlgorithm miner algorithm ethash
	EthashAlgorithm = "ethash"

	// Sha256Algorithm miner algorithm sha256
	Sha256Algorithm = "sha256"

	// spow miner algorithm
	SpowAlgorithm = "spow"

	// BFT mineralgorithm
	BFTEngine = "bft"

	// BFT data folder
	BFTDataFolder = "bftdata"

	// EVMStackLimit increase evm stack limit to 8192
	EVMStackLimit = 8192

	// BlockPackInterval it's an estimate time.
	BlockPackInterval = 15 * time.Second

	// Height: fix the issue caused by forking from collapse database
	HeightFloor = uint64(707989)
	HeightRoof  = uint64(707996)

	WindowsPipeDir = `\\.\pipe\`

	defaultPipeFile = `\seele.ipc`
)

var (
	// tempFolder used to store temp file, such as log files
	tempFolder string

	// defaultDataFolder used to store persistent data info, such as the database and keystore
	defaultDataFolder string

	// defaultIPCPath used to store the ipc file
	defaultIPCPath string
)

// Common big integers often used
var (
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(2)
	Big3   = big.NewInt(3)
	Big0   = big.NewInt(0)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(256)
	Big257 = big.NewInt(257)
)

func init() {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	tempFolder = filepath.Join(usr.HomeDir, "seeleTemp")

	defaultDataFolder = filepath.Join(usr.HomeDir, ".seele")

	if runtime.GOOS == "windows" {
		defaultIPCPath = WindowsPipeDir + defaultPipeFile
	} else {
		defaultIPCPath = filepath.Join(defaultDataFolder, defaultPipeFile)
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
