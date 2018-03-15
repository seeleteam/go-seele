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
	//Fatal("fatal msg")
	//panic("panic msg")
}

func Test_LogFile(t *testing.T) {
	log := GetLogger("test2", false)

	log.Debug("debug")

	exist := common.IsFileOrFolderExist(filepath.Join(logFolder, "test2.log"))
	assert.Equal(t, exist, true)
}
