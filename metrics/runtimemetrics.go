/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package metrics

import (
	"runtime"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"os"
	"fmt"
	"bufio"
	"io"
	"strconv"
	"strings"
)

var refresh = 5 * time.Second

// CollectRuntimeMetrics collected runtime datas
func collectRuntimeMetrics() {
	if metrics.UseNilMetrics {
		return
	}

	memAlloc := metrics.GetOrRegisterGauge("runtime.memory.alloc", metrics.DefaultRegistry)
	memPauses := metrics.GetOrRegisterMeter("runtime.memory.pauses", metrics.DefaultRegistry)

	memStats := new(runtime.MemStats)
	var lastPauseNs uint64
	// collect metrics
	for {
		runtime.ReadMemStats(memStats)
		memAlloc.Update(int64(memStats.Alloc))
		memPauses.Mark(int64(memStats.PauseTotalNs - lastPauseNs))
		lastPauseNs = memStats.PauseTotalNs

		result,err := getCPU(refresh, false)
		fmt.Printf("========cup Mark: %v\n", result)
		if err == nil{
			metricsCputGauge.Update(result)
			fmt.Printf("========cup Mark: %v\n", result)
		}
		// sleep 5 seconds
		time.Sleep(refresh)
	}
}

// DiskStats is the per process disk io stats.
type DiskStats struct {
	ReadCount  int64 // Number of read operations executed
	ReadBytes  int64 // Total number of bytes read
	WriteCount int64 // Number of write operations executed
	WriteBytes int64 // Total number of byte written
}

// ReadDiskStats retrieves the disk IO stats belonging to the current process.
func ReadDiskStats(stats *DiskStats) error {
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
