package metrics

import (
	metrics "github.com/rcrowley/go-metrics"
	"strconv"
	"os"
	"fmt"
	"bufio"
	"io"
	"strings"
	"runtime"
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
		if err := GetDiskInfoLinux(&diskStats); err != nil{
			return &diskStats
		}
	}
	return nil
}

// GetDiskInfo retrieves the disk IO info belonging to the current process.
func GetDiskInfoLinux(stats *DiskStats) error {
	// Open the process disk IO counter file
	inf, err := os.Open(fmt.Sprintf("/proc/%d/io", os.Getpid()))
	if err != nil {
		return err
	}
	defer inf.Close()
	in := bufio.NewReader(inf)

	// Iterate over the IO counter, and extract what we need
	for {
		// Read the next line and split to key and value
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

		// Update the counter based on the key
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
