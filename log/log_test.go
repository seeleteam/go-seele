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
	"github.com/sirupsen/logrus"
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

func Test_LogLevels(t *testing.T) {
	log := GetLogger("test3", true)
	levels := log.GetLevels()
	fmt.Println("levels:", levels)
	for i, v := range levels {
		fmt.Println("i:", i, ",v:", uint32(v))
	}
	level := log.GetLevel()
	fmt.Println("level:", level)
	log.SetLogLevel(logrus.DebugLevel)
	level = log.GetLevel()
	fmt.Println("changed level is:", level)
	log.Debug("debug can be output")
	log.Warn("warning can be output")
	log.Info("Info can be output")
	log.SetLogLevel(logrus.WarnLevel)
	level = log.GetLevel()
	fmt.Println("changed level is:", level)
	log.Debug("debug can be output")
	log.Warn("warning can be output")
	log.Info("Info can be output")
}
