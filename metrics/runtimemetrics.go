/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package metrics

import (
	"runtime"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/common"
	"github.com/shirou/gopsutil/disk"
)

var (
	memAlloc  = metrics.GetOrRegisterGauge("runtime.memory.alloc", metrics.DefaultRegistry)
	memPauses = metrics.GetOrRegisterMeter("runtime.memory.pauses", metrics.DefaultRegistry)
)

// CollectRuntimeMetrics collected runtime datas
func collectRuntimeMetrics() {
	if metrics.UseNilMetrics {
		return
	}

	memStats := new(runtime.MemStats)
	var lastPauseNs uint64
	// collect metrics
	for {
		doMark(memStats, lastPauseNs)
		time.Sleep(common.MetricsRefreshTime)
	}
}

func doMark(memStats *runtime.MemStats, lastPauseNs uint64) {
	runtime.ReadMemStats(memStats)
	memAlloc.Update(int64(memStats.Alloc))
	memPauses.Mark(int64(memStats.PauseTotalNs - lastPauseNs))
	lastPauseNs = memStats.PauseTotalNs

	// cpuResult is the cpu info of the current system
	if cpuResult, err := getCPURate(common.CPUMetricsRefreshTime, false); err == nil {
		metricsCpuGauge.Update(cpuResult)
	}

	// cpuSeeleResult is the cpu info of the current process
	if cpuSeeleResult, err := getProcessCPURate(common.CPUMetricsRefreshTime); err == nil {
		metricsSeeleCpuGauge.Update(cpuSeeleResult)
	}

	// diskResult is the disk info of the current system
	if diskResult, err := disk.Usage(common.GetDefaultDataFolder()); err == nil {
		metricsDiskFreeCountGauge.Update(int64(diskResult.Free))
	}

	// diskInfo is the disk info of the current process
	if diskInfo := GetDiskInfo(); diskInfo != nil {
		metricsDiskReadCountGauge.Update(int64(diskInfo.ReadCount))
		metricsDiskWriteCountGauge.Update(int64(diskInfo.WriteCount))
		metricsDiskReadBytesGauge.Update(int64(diskInfo.ReadBytes))
		metricsDiskWriteBytesGauge.Update(int64(diskInfo.WriteBytes))
	}
}
