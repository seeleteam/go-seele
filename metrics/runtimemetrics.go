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

		cpuresult, err := getCPURate(common.RefreshTime, false)
		if err == nil {
			metricsCputGauge.Update(cpuresult)
		}

		diskresult, err := disk.Usage(common.GetDefaultDataFolder())
		if err == nil {
			metricsDiskUsedCountGauge.Update(int64(diskresult.Used))
			metricsDiskFreeCountGauge.Update(int64(diskresult.Free))
			metricsDiskUsedPercentGauge.Update(int64(diskresult.UsedPercent))
			metricsDiskTotalCountGauge.Update(int64(diskresult.Total))
		}
		// sleep 5 seconds
		time.Sleep(common.RefreshTime)
	}
}
