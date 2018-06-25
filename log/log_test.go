/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

import (
	"path/filepath"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
)

func Test_Log(t *testing.T) {
	lg := GetLogger("test", true)
	lg.Debug("debug msg")
	lg.Info("info msg")
	lg.Warn("warn msg")
	lg.Error("error msg")
	lg.Info("fold is:", LogFolder)
	//Fatal("fatal msg")
	//panic("panic msg")
}

func Test_LogFile(t *testing.T) {
	log := GetLogger("test2", false)

	log.Debug("debug")
	log.Info("info msg")
	log.Warn("warn msg")
	log.Error("error msg")
	log.Info("fold is:", LogFolder)
	log.Info("log file is:%s/", LogFolder, common.LogFileName)
	exist := common.FileOrFolderExists(filepath.Join(LogFolder, common.LogFileName))
	assert.Equal(t, exist, true)
}
