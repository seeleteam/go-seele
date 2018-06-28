package metrics

import (
	"os"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
)

var (
	metricsCputGauge      = metrics.GetOrRegisterGauge("cpu.accout", nil)
	metricsSeeleCputGauge = metrics.GetOrRegisterGauge("cpu.seele.accout", nil)
)

// getCPURate get the CPU percent that the current system has already used
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

// getProcessCPURate get the CPU percent that the current process has already used
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
