/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

import (
	"testing"
)

func Test_Log(t *testing.T) {
	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")
	//Fatal("fatal msg")
	//panic("panic msg")
}
