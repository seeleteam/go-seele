/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

import (
	"fmt"
	"github.com/seeleteam/go-seele/common"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"sync"
)

var (
	// LogFolder the default folder to write logs
	LogFolder = filepath.Join(common.GetTempFolder(), "Log")
)

// SeeleLog wraps log class
type SeeleLog struct {
	log    *logrus.Logger
	level  logrus.Level
	module string
}

var logMap map[string]*SeeleLog
var getLogMutex sync.Mutex

//Default exported log tag for all users
var Default *SeeleLog

// NewSeeleLog create a pointer of SeeleLog
func NewSeeleLog() *SeeleLog {
	return &SeeleLog{
		log:    logrus.New(),
		level:  logrus.InfoLevel,
		module: "log",
	}
}
func init() {
	Default = NewSeeleLog()
	logFullPath := filepath.Join(LogFolder, "log.log")
	file, err := os.OpenFile(logFullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(fmt.Sprintf("creating log file failed: %s", err.Error()))
	}
	Default.log.Out = file
	Default.log.AddHook(&CallerHook{})
}

//SetMod setting module tags for information
func (p *SeeleLog) SetMod(module string) {
	p.module = module
}

//GetMod setting module tags for information
func (p *SeeleLog) GetMod() string {
	return p.module
}

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

// GetLevel obtain the current level station
func (p *SeeleLog) GetLevel() logrus.Level {
	return p.level
}

//SetLevel set the current level station
func (p *SeeleLog) SetLevel(level logrus.Level) {
	p.level = level
	p.log.SetLevel(level)
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
			panic(fmt.Sprintf("creating log file failed: %s", err.Error()))
		}

		logFileName := logName + ".log"
		logFullPath := filepath.Join(LogFolder, logFileName)
		file, err := os.OpenFile(logFullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			panic(fmt.Sprintf("creating log file failed: %s", err.Error()))
		}

		log.Out = file
	}

	if common.IsDebug {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	log.AddHook(&CallerHook{}) // add caller hook to print caller's file and line number
	curLog = &SeeleLog{
		log: log,
	}
	logMap[logName] = curLog
	return curLog
}
