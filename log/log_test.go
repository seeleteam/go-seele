/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

import (
	"testing"
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
