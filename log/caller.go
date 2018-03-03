package log

import (
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	depth = 9 // Once log invocation stack changed, depth need to change as well.
)

// CallerHook a caller hook of logrus
type CallerHook struct {
}

// Fire adds a callers field in logger instance
func (hook *CallerHook) Fire(entry *logrus.Entry) error {
	entry.Data["caller"] = hook.caller()
	return nil
}

// Levels returns support levels
func (hook *CallerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *CallerHook) caller() string {
	if _, file, line, ok := runtime.Caller(depth); ok {
		return strings.Join([]string{filepath.Base(file), strconv.Itoa(line)}, ":")
	}

	// not sure what the convention should be here
	return ""
}
