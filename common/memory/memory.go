/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package memory

import (
	"fmt"
	"runtime"
	"time"

	"github.com/seeleteam/go-seele/log"
)

const (
	prefix string = "memory info collection"
)

// GetMemoryInfo return current memory information
func GetMemoryInfo() runtime.MemStats {
	var info runtime.MemStats
	runtime.ReadMemStats(&info)

	return info
}

// Print is used to print log
func Print(p *log.SeeleLog, msg string, t time.Time, isCalTime bool) {
	memInfo := GetMemoryInfo()
	if isCalTime {
		p.Debug(fmt.Sprint(prefix, ", ", msg, ", alloc %.4fGB, sys %.4fGB, time elapse %.2fs"), BToGB(memInfo.Alloc), BToGB(memInfo.Sys), time.Since(t).Seconds())
	} else {
		p.Debug(fmt.Sprint(prefix, ", ", msg, ", alloc %.4fGB, sys %.4fGB"), BToGB(memInfo.Alloc), BToGB(memInfo.Sys))
	}
}

// BToGB bytes to GB
func BToGB(n uint64) float64 {
	return float64(n) / 1024 / 1024 / 1024
}
