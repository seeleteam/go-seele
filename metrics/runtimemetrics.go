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

	memAllocs := metrics.GetOrRegisterGauge("runtime.memory.allocs", metrics.DefaultRegistry)
	memFrees := metrics.GetOrRegisterGauge("runtime.memory.frees", metrics.DefaultRegistry)

	memStats := new(runtime.MemStats)
	// collect metrics
	for {
		runtime.ReadMemStats(memStats)
		memAllocs.Update(int64(memStats.Mallocs))
		memFrees.Update(int64(memStats.Frees))

		// sleep 5 seconds
		time.Sleep(5 * time.Second)
	}
}
