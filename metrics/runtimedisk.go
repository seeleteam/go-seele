package metrics

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	metrics "github.com/rcrowley/go-metrics"
)

var (
	metricsDiskFreeCountGauge  = metrics.GetOrRegisterGauge("disk.free.count", nil)
	metricsDiskReadCountGauge  = metrics.GetOrRegisterGauge("disk.read.count", nil)
	metricsDiskWriteCountGauge = metrics.GetOrRegisterGauge("disk.write.count", nil)
	metricsDiskReadBytesGauge  = metrics.GetOrRegisterGauge("disk.read.bytes", nil)
	metricsDiskWriteBytesGauge = metrics.GetOrRegisterGauge("disk.write.bytes", nil)
)

type DiskStats struct {
	ReadCount  int64
	ReadBytes  int64
	WriteCount int64
	WriteBytes int64
}

func GetDiskInfo() *DiskStats {
	diskStats := DiskStats{}
	if runtime.GOOS == "linux" {
		if err := getDiskInfoLinux(&diskStats); err == nil {
			return &diskStats
		}
	}
	return nil
}

// getDiskInfoLinux retrieves the disk IO info belonging to the current process.
func getDiskInfoLinux(stats *DiskStats) error {
	inf, err := os.Open(fmt.Sprintf("/proc/%d/io", os.Getpid()))
	if err != nil {
		return err
	}
	defer inf.Close()
	in := bufio.NewReader(inf)

	for {
		line, err := in.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil {
			return err
		}

		switch key {
		case "syscr":
			stats.ReadCount = value
		case "syscw":
			stats.WriteCount = value
		case "rchar":
			stats.ReadBytes = value
		case "wchar":
			stats.WriteBytes = value
		}
	}
}
