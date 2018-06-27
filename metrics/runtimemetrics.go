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
)

var refreshTime = 5 * time.Second

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

		cpuresult,err := getCPURate(refreshTime, false)
		if err == nil{
			metricsCputGauge.Update(cpuresult)
		}

		diskresult,err := getDiskRate(common.GetTempFolder())
		if err == nil{
			metricsDiskReadCountGauge.Update(int64(diskresult.ReadCount))
			metricsDiskReadBytesGauge.Update(int64(diskresult.ReadBytes))
			metricsDiskWriteCountGauge.Update(int64(diskresult.WriteCount))
			metricsDiskWriteBytesGauge.Update(int64(diskresult.WriteBytes))
		}
		// sleep 5 seconds
		time.Sleep(refreshTime)
	}
}
