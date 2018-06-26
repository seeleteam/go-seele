/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"time"
	"path/filepath"
	"github.com/magiconair/properties/assert"
)

func Test_Log(t *testing.T) {
	lg := GetLogger("test", true)
	lg.Debug("debug msg")
	lg.Info("info msg")
	lg.Warn("warn msg")
	lg.Error("error msg")
	lg.Info("folder is: %s", LogFolder)
}

func Test_LogFile(t *testing.T) {
	log := GetLogger("test2", false)

	log.Debug("debug")
	log.Info("info msg")
	log.Warn("warn msg")

	log.Error("error msg")
	log.Info("folder is:", LogFolder)

	now := time.Now().Format(".20060102")
	logPath := filepath.Join(LogFolder, common.LogFileName) + now
	log.Info("log file is:%s", logPath)

	exist := common.FileOrFolderExists(logPath)
	assert.Equal(t, exist, true)
}
