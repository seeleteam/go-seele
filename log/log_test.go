/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/sirupsen/logrus"
)

func Test_Log(t *testing.T) {
	lg := GetLogger("test", true)
	lg.Debug("debug msg")
	lg.Info("info msg")
	lg.Warn("warn msg")
	lg.Error("error msg")
	lg.Info("folder is: %s", LogFolder)

	newLg := GetLogger("test", true)
	assert.Equal(t, lg, newLg)
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

func Test_LogLevels(t *testing.T) {
	log := GetLogger("test3", true)
	log.SetLevel(logrus.InfoLevel)
	log.Debug("debug can be done")
	log.Info("Info can be done")
	log.Warn("Warn can be done")
	assert.Equal(t, logrus.InfoLevel, log.GetLevel())

	// Default is DebugLevel due to common.LogConfig.IsDebug is true
	log = GetLogger("test4", true)
	assert.Equal(t, logrus.DebugLevel, log.GetLevel())

	// Set common.LogConfig.IsDebug as false
	isDebug := common.LogConfig.IsDebug
	defer func() {
		common.LogConfig.IsDebug = isDebug
	}()

	common.LogConfig.IsDebug = false
	log = GetLogger("test5", true)
	assert.Equal(t, logrus.InfoLevel, log.GetLevel())
}
