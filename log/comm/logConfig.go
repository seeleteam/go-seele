/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package comm

var (
	// LogExtension default log file extension
	LogExtension = "log"

	// LogPrefix default log file prefix
	LogPrefix = "seele"
)

// LogConfig is the Configuration of log
type LogConfig struct {
	// If IsDebug is true, the log level will be DebugLevel, otherwise it is InfoLevel
	IsDebug bool `json:"isDebug"`

	// If PrintLog is true, all logs will be printed in the console, otherwise they will be stored in the file.
	PrintLog bool `json:"printLog"`
}
