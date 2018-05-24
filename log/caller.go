package log

import (
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	depth = 9 // Once log invocation stack has changed, depth needs to change as well.
)

// CallerHook a caller hook of logrus
type CallerHook struct {
	module string
}

// Fire adds a caller field in logger instance
func (hook *CallerHook) Fire(entry *logrus.Entry) error {
	entry.Data["caller"] = hook.caller()
	entry.Data["module"] = hook.module
	return nil
}

// Levels returns supported levels
func (hook *CallerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// caller returns the invoker which is being executed
func (hook *CallerHook) caller() string {
	if _, file, line, ok := runtime.Caller(depth); ok {
		return strings.Join([]string{filepath.Base(file), strconv.Itoa(line)}, ":")
	}

	// not sure what the convention should be here
	return ""
}
