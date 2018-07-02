package core

import metrics "github.com/rcrowley/go-metrics"

var metricsWriteBlockMeter = metrics.GetOrRegisterMeter("write.block.time", nil)
