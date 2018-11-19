/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
)

var (
	lastLogTime = time.Now()
)

func log(format string, a ...interface{}) {
	now := time.Now()
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%v | %v (elapsed time: %v)\n", now.Format("2006-01-02 15:04:05"), msg, now.Sub(lastLogTime))
	lastLogTime = now
}

func prepareDir(dir string) error {
	var err error
	if dir, err = filepath.Abs(dir); err != nil {
		return errors.NewStackedErrorf(err, "failed to get abs path for %v", dir)
	}

	if common.FileOrFolderExists(dir) {
		if err = os.RemoveAll(dir); err != nil {
			return errors.NewStackedErrorf(err, "failed to delete folder %v before test", dir)
		}
	}

	if err = os.MkdirAll(dir, os.ModePerm); err != nil {
		return errors.NewStackedErrorf(err, "failed to create folder %v recursively before test", dir)
	}

	return nil
}

func getLevelDBSize(rootDir, dbName string) uint64 {
	dbPath := filepath.Join(rootDir, dbName)

	files, err := ioutil.ReadDir(dbPath)
	if err != nil {
		panic(errors.NewStackedErrorf(err, "failed to read dir %v", dbPath))
	}

	var totalSize int64
	for _, f := range files {
		if f.IsDir() {
			panic(errors.NewStackedErrorf(err, "unexpected folder %v", f.Name()))
		}

		totalSize += f.Size()
	}

	return uint64(totalSize)
}

func sizeToString(size uint64) string {
	if size < 1024 {
		return fmt.Sprintf("%v Bytes", size)
	}

	if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024.0)
	}

	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/1024.0/1024.0)
	}

	if size < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.2f GB", float64(size)/1024.0/1024.0/1024.0)
	}

	if size < 1024*1024*1024*1024*1024 {
		return fmt.Sprintf("%.2f TB", float64(size)/1024.0/1024.0/1024.0/1024.0)
	}

	return fmt.Sprint(size)
}
