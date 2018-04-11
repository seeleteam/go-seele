/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/sirupsen/logrus"
)

var (
	// LogFolder the default folder to write log
	LogFolder = filepath.Join(common.GetTempFolder(), "Log")
)

// SeeleLog wrapped log class
type SeeleLog struct {
	log *logrus.Logger
}

var logMap map[string]*SeeleLog
var getLogMutex sync.Mutex

// Panic Level, highest level of severity. Logs and then calls panic with the
// message passed to Debug, Info, ...
func (p *SeeleLog) Panic(format string, args ...interface{}) {
	p.log.Panicf(format, args...)
}

// Fatal Level. Logs and then calls `os.Exit(1)`. It will exit even if the
// logging level is set to Panic.
func (p *SeeleLog) Fatal(format string, args ...interface{}) {
	p.log.Fatalf(format, args...)
}

// Error Level. Logs. Used for errors that should definitely be noted.
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

// GetLogger get logrus.Logger object accoring to logName
// each module can have it's own logger
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
			panic(fmt.Sprintf("create log file failed %s", err))
		}

		logFileName := logName + ".log"
		logFullPath := filepath.Join(LogFolder, logFileName)
		file, err := os.OpenFile(logFullPath, os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(fmt.Sprintf("create log file failed %s", err))
		}

		log.Out = file
	}

	if common.IsDebug {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	log.AddHook(&CallerHook{}) // add caller hook to print caller's file & line number
	curLog = &SeeleLog{
		log: log,
	}
	logMap[logName] = curLog
	return curLog
}
