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
	// LogFolder the default folder to write logs
	LogFolder = filepath.Join(common.GetTempFolder(), "Log")
)

// LogFile is the file which records all logs by users
const LogFile string = "log.log"

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
			panic(fmt.Sprintf("creating log dir failed: %s", err.Error()))
		}
		logFullPath := filepath.Join(LogFolder, LogFile)
		if _, err1 := os.Stat(logFullPath); err1 != nil {
			if os.IsNotExist(err1) {
				cfile, cerr := os.Create(logFullPath)
				if cerr != nil {
					panic(fmt.Sprintf("creating log file failed: %s", cerr.Error()))
				}
				cfile.Chmod(os.ModeAppend)
				log.Out = cfile
			}
		} else {
			os.Chmod(logFullPath, os.ModePerm)
			file, err2 := os.OpenFile(logFullPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
			if err2 != nil {
				panic(fmt.Sprintf("open %s file failed: %s", logFullPath, err2.Error()))
			}
			log.Out = file
		}

		// file, err := os.OpenFile(logFullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		// if err != nil {
		// 	panic(fmt.Sprintf("creating log file failed: %s", err.Error()))
		// }

	}

	if common.IsDebug {
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
