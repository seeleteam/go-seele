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

		cpuResult, err := getCPURate(common.RefreshTime, false)
		if err == nil {
			metricsCputGauge.Update(cpuResult)
		}
		cpuSeeleResult, err := getProcessCPURate(common.RefreshTime)
		if err == nil {
			metricsSeeleCputGauge.Update(cpuSeeleResult)
		}

		diskResult, err := disk.Usage(common.GetDefaultDataFolder())
		if err == nil {
			metricsDiskFreeCountGauge.Update(int64(diskResult.Free))
		}

		if diskInfo := GetDiskInfo(common.GetDefaultDataFolder()); diskInfo != nil {
			metricsDiskReadCountGauge.Update(int64(diskInfo.ReadCount))
			metricsDiskWriteCountGauge.Update(int64(diskInfo.WriteCount))
			metricsDiskReadBytesGauge.Update(int64(diskInfo.ReadBytes))
			metricsDiskWriteBytesGauge.Update(int64(diskInfo.WriteBytes))
			metricsDiskIoTimeGauge.Update(int64(diskInfo.IoTime))
		}

		time.Sleep(common.RefreshTime)
	}
}
