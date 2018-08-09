/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/seeleteam/go-seele/common"
	"github.com/sirupsen/logrus"
)

var (
	// LogFolder the default folder to write logs
	LogFolder = filepath.Join(common.GetTempFolder(), "log")
)

// SeeleLog wraps log class
type SeeleLog struct {
	log *logrus.Logger
}

var logMap map[string]*SeeleLog
var getLogMutex sync.Mutex

// Panic Level, highest level of severity. Panic logs and then calls panic with the
// message passed to Debug, Info, ...
func (p *SeeleLog) Panic(format string, args ...interface{}) {
	p.log.Panicf(format, args...)
}

// Fatal Level. Fatal logs and then calls `os.Exit(1)`. It will exit even if the
// logging level is set to Panic.
func (p *SeeleLog) Fatal(format string, args ...interface{}) {
	p.log.Fatalf(format, args...)
}

// Error Level. Error logs and is used for errors that should be definitely noted.
// Commonly used for hooks to send errors to an error tracking service.
func (p *SeeleLog) Error(format string, args ...interface{}) {
	p.log.Errorf(format, args...)
}

// Warn Level. Non-critical entries that deserve eyes.
func (p *SeeleLog) Warn(format string, args ...interface{}) {
	p.log.Warnf(format, args...)
}

// Info Level. General operational entries about what's going on inside the
// application.
func (p *SeeleLog) Info(format string, args ...interface{}) {
	p.log.Infof(format, args...)
}

// Debug Level. Usually only enabled when debugging. Very verbose logging.
func (p *SeeleLog) Debug(format string, args ...interface{}) {
	p.log.Debugf(format, args...)
}

// SetLevel set the log level
func (p *SeeleLog) SetLevel(level logrus.Level) {
	p.log.SetLevel(level)
}

// GetLevel get the log level
func (p *SeeleLog) GetLevel() logrus.Level {
	return p.log.Level
}

// GetLogger gets logrus.Logger object according to logName
// each module can have its own logger
func GetLogger(logName string, bConsole bool) *SeeleLog {
	getLogMutex.Lock()
	defer getLogMutex.Unlock()
	if logMap == nil {
		logMap = make(map[string]*SeeleLog)
	}
	curLog, ok := logMap[logName]
	if ok {
		return curLog
	}

	logrus.SetFormatter(&logrus.TextFormatter{})
	log := logrus.New()

	if bConsole {
		log.Out = os.Stdout
	} else {
		err := os.MkdirAll(LogFolder, os.ModePerm)
		if err != nil {
			panic(fmt.Sprintf("failed to create log dir: %s", err.Error()))
		}
		logFullPath := filepath.Join(LogFolder, common.LogFileName)

		ext := filepath.Ext(logFullPath)
		writer, err := rotatelogs.New(
			logFullPath[:strings.Index(logFullPath, ext)]+".%Y%m%d"+ext,
			rotatelogs.WithClock(rotatelogs.Local),
			rotatelogs.WithMaxAge(24*7*time.Hour),
			rotatelogs.WithRotationTime(24*time.Hour),
		)

		if err != nil {
			panic(fmt.Sprintf("failed to create log file: %s", err))
		}

		log.Out = writer
	}

	if common.LogConfig.IsDebug {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	log.AddHook(&CallerHook{module: logName}) // add caller hook to print caller's file and line number
	curLog = &SeeleLog{
		log: log,
	}
	logMap[logName] = curLog
	return curLog
}
