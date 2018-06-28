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

		if cpuResult, err := getCPURate(common.RefreshTime, false); err == nil {
			metricsCputGauge.Update(cpuResult)
		}

		if cpuSeeleResult, err := getProcessCPURate(common.RefreshTime); err == nil {
			metricsSeeleCputGauge.Update(cpuSeeleResult)
		}

		if diskResult, err := disk.Usage(common.GetDefaultDataFolder()); err == nil {
			metricsDiskFreeCountGauge.Update(int64(diskResult.Free))
		}

		if diskInfo := GetDiskInfo(); diskInfo != nil {
			metricsDiskReadCountGauge.Update(int64(diskInfo.ReadCount))
			metricsDiskWriteCountGauge.Update(int64(diskInfo.WriteCount))
			metricsDiskReadBytesGauge.Update(int64(diskInfo.ReadBytes))
			metricsDiskWriteBytesGauge.Update(int64(diskInfo.WriteBytes))
		}

		time.Sleep(common.RefreshTime)
	}
}
