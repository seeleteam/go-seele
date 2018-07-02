package metrics

import (
	"os"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
)

var (
	// metricsCpuGauge is the usage rate of the metrics current system
	metricsCpuGauge = metrics.GetOrRegisterGauge("cpu.os", nil)
	// metricsSeeleCpuGauge is the usage rate of the metrics current process
	metricsSeeleCpuGauge = metrics.GetOrRegisterGauge("cpu.seele", nil)
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

func getProcessCPURate(interval time.Duration) (int64, error) {
	checkPid := os.Getpid()
	ret, err := process.NewProcess(int32(checkPid))
	if err != nil {
		return 0, err
	}

	result, err := ret.Percent(interval)
	if err != nil {
		return 0, err
	}
	return int64(result), nil
}
