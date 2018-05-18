/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

// Config is the Configuration of log
type Config struct {

	// If IsDebug is true, the log level will be DebugLevel, otherwise it is InfoLevel
	IsDebug bool `json:"isDebug"`

	// If PrintLog is true, all logs will be printed in the console, otherwise they will be stored in the file.
	PrintLog bool `json:"printLog"`
}
