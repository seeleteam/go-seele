/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/sirupsen/logrus"
)

func Test_Log(t *testing.T) {
	lg := GetLogger("test")
	lg.Debug("debug msg")
	lg.Info("info msg")
	lg.Warn("warn msg")
	lg.Error("error msg")
	lg.Info("folder is: %s", LogFolder)

	newLg := GetLogger("test")
	assert.Equal(t, lg, newLg)
}

func Test_LogFile(t *testing.T) {
	originPrintLog := comm.Config.PrintLog
	defer func() {
		comm.Config.PrintLog = originPrintLog
	}()
	comm.Config.PrintLog = false
	log := GetLogger("test2")

	log.Debug("debug")
	log.Info("info msg")
	log.Warn("warn msg")

	log.Error("error msg")
	log.Info("folder is:", LogFolder)

	now := time.Now().Format("20060102")
	logFileName := fmt.Sprintf("%s.%s%s", comm.Config.LogFilePrefix, now, logExtension)
	logPath := filepath.Join(LogFolder, logFileName)

	log.Info("log file is:%s", logPath)

	exist := common.FileOrFolderExists(logPath)
	assert.Equal(t, exist, true)
}

func Test_LogLevels(t *testing.T) {
	log := GetLogger("test3")
	log.SetLevel(logrus.InfoLevel)
	log.Debug("debug can be done")
	log.Info("Info can be done")
	log.Warn("Warn can be done")
	assert.Equal(t, logrus.InfoLevel, log.GetLevel())

	// Default is DebugLevel due to comm.Config.IsDebug is true
	log = GetLogger("test4")
	assert.Equal(t, logrus.DebugLevel, log.GetLevel())

	// Set ccomm.Config.IsDebug as false
	isDebug := comm.Config.IsDebug
	defer func() {
		comm.Config.IsDebug = isDebug
	}()

	comm.Config.IsDebug = false
	log = GetLogger("test5")
	assert.Equal(t, logrus.InfoLevel, log.GetLevel())
}
