/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package log

import (
    "fmt"
    "github.com/sirupsen/logrus"
    "os"
    "os/user"
    "path/filepath"
)

var log *logrus.Logger

// Panic Level, highest level of severity. Logs and then calls panic with the
// message passed to Debug, Info, ...
func Panic(args ...interface{}) {
    log.Panicln(args)
}

// Fatal Level. Logs and then calls `os.Exit(1)`. It will exit even if the
// logging level is set to Panic.
func Fatal(args ...interface{})  {
    log.Fatal(args)
}

// Error Level. Logs. Used for errors that should definitely be noted.
// Commonly used for hooks to send errors to an error tracking service.
func Error(args ...interface{}) {
    log.Errorln(args)
}

// Warn Level. Non-critical entries that deserve eyes.
func Warn(args ...interface{}) {
    log.Warn(args)
}

// Info Level. General operational entries about what's going on inside the
// application.
func Info(args ...interface{}) {
    log.Infoln(args)
}

// Debug Level. Usually only enabled when debugging. Very verbose logging.
func Debug(args ...interface{}) {
    log.Debugln(args)
}

func init()  {
    usr, err := user.Current()
    if err != nil {
        fmt.Println("can't get current user info. ", err)
        return
    }

    seeleFolder := filepath.Join(usr.HomeDir, "seelelog")
    if _, err := os.Stat(seeleFolder); err != nil {
        if err = os.Mkdir(seeleFolder, os.ModeDir); err != nil {
            fmt.Println("create log folder failed. ", err)
            return
        }
    }

    seeleLog := filepath.Join(seeleFolder, "log.txt")
    file, err := os.OpenFile(seeleLog, os.O_CREATE | os.O_WRONLY, 0666)
    if err != nil {
        fmt.Println("create log file failed. ", err)
        return
    }

    logrus.SetFormatter(&logrus.TextFormatter{})

    log = logrus.New()
    //log.Out = file //use std out for temp
    log.Out = os.Stdout
    log.SetLevel(logrus.DebugLevel)
}