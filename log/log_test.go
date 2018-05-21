/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

import (
	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"path/filepath"
	"testing"
)

func Test_Log_File(t *testing.T) {
	lg := GetLogger("test", false)
	lg.Debug("debug msg")
	lg.Info("info msg")
	lg.Info(filepath.Join(common.GetTempFolder(), "Log"))
	lg.Warn("warn msg")
	lg.Error("error msg")
	Loging.Info("I am in testing you are best!!!")
	//Fatal("fatal msg")
	//panic("panic msg")
}

func Test_Log_Terminal(t *testing.T) {
	lg := GetLogger("test", true)
	lg.Debug("debug msg")
	lg.Info("info msg")
	lg.Info(filepath.Join(common.GetTempFolder(), "Log"))
	lg.Warn("warn msg")
	lg.Error("error msg")
	//Fatal("fatal msg")
	//panic("panic msg")
}

func Test_LogFile(t *testing.T) {
	log := GetLogger("test2", false)

	log.Debug("debug")

	exist := common.FileOrFolderExists(filepath.Join(LogFolder, "test2.log"))
	assert.Equal(t, exist, true)
}
