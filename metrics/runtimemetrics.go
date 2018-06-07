/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package metrics

import (
	"runtime"
	"time"

	metrics "github.com/rcrowley/go-metrics"
)

// CollectRuntimeMetrics collected runtime datas
func collectRuntimeMetrics() {
	if metrics.UseNilMetrics {
		return
	}

	memAlloc := metrics.GetOrRegisterGauge("runtime.memory.alloc", metrics.DefaultRegistry)
	memPauses := metrics.GetOrRegisterGauge("runtime.memory.pauses", metrics.DefaultRegistry)

	memStats := new(runtime.MemStats)
	// collect metrics
	for {
		runtime.ReadMemStats(memStats)
		memAlloc.Update(int64(memStats.Alloc))
		memPauses.Update(int64(memStats.PauseTotalNs))

		// sleep 5 seconds
		time.Sleep(5 * time.Second)
	}
}
