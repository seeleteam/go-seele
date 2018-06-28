package metrics

import (
	metrics "github.com/rcrowley/go-metrics"
)

var (
	metricsDiskUsedCountGauge  = metrics.GetOrRegisterGauge("disk.used.count", nil)
	metricsDiskFreeCountGauge  = metrics.GetOrRegisterGauge("disk.free.count", nil)
	metricsDiskUsedPercentGauge = metrics.GetOrRegisterGauge("disk.used.percent", nil)
	metricsDiskTotalCountGauge = metrics.GetOrRegisterGauge("disk.total.count", nil)
)
