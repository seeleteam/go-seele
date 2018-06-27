package metrics

import (
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/shirou/gopsutil/cpu"
)

var (
	metricsCputGauge = metrics.GetOrRegisterGauge("cpu.accout", nil)
)

func getCPURate(interval time.Duration, percpu bool) (int64, error) {
	out, err := cpu.Percent(interval, percpu)
	if err != nil {
		return 0, err
	}
	var result float64
	for i := 0; i < len(out); i++ {
		result = result + out[i]
	}
	return int64(result), nil
}
