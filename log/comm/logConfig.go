/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package comm

// Config is the Configuration of log
var Config = &LogConfig{PrintLog: true, IsDebug: true, LogFilePrefix: "seele"}

// LogConfig is the Configuration of log
type LogConfig struct {
	// If IsDebug is true, the log level will be DebugLevel, otherwise it is InfoLevel
	IsDebug bool `json:"isDebug"`

	// If PrintLog is true, all logs will be printed in the console, otherwise they will be stored in the file.
	PrintLog bool `json:"printLog"`

	// LogPrefix default log file prefix
	LogFilePrefix string `json:"logFilePrefix"`
}
